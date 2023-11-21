// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workloadidentity

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1clientset "github.com/gardener/gardener/pkg/client/authentication/clientset/versioned/typed/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
)

const (
	defaultValidityDuration = 12 * time.Hour
)

// Reconciler requests and refreshes tokens via the Workload Identity TokenRequest API.
type Reconciler struct {
	GardenClient        client.Client
	SeedClient          client.Client
	GardenAuthClientset *authenticationv1alpha1clientset.AuthenticationV1alpha1Client
	Clock               clock.Clock
	JitterFunc          func(duration time.Duration, maxFactor float64) time.Duration
	SeedName            string
}

// Reconcile requests and populates tokens.
func (r *Reconciler) Reconcile(reconcileCtx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(reconcileCtx)

	ctx, cancel := controllerutils.GetMainReconciliationContext(reconcileCtx, controllerutils.DefaultReconciliationTimeout)
	defer cancel()

	shoot := &gardencorev1beta1.Shoot{}
	if err := r.GardenClient.Get(ctx, req.NamespacedName, shoot); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object is gone, stop reconciling", "shoot", req.NamespacedName.String())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if !isRelevantShoot(shoot) {
		return reconcile.Result{}, nil
	}

	wi := &authenticationv1alpha1.WorkloadIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shoot.Labels["workloadidentity"], // TODO fix this
			Namespace: req.NamespacedName.Namespace,
		},
	}

	if err := r.GardenClient.Get(ctx, client.ObjectKeyFromObject(wi), wi); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object is gone, stop reconciling", "workloadIdentity", client.ObjectKeyFromObject(wi).String())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wi--" + wi.Name,
			Namespace: shoot.Status.TechnicalID,
		},
	}

	if err := r.SeedClient.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		// if the secret does not exist we will create it later
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}

	mustRequeue, requeueAfter, err := r.requeue(ctx, secret)
	if err != nil {
		return reconcile.Result{}, err
	}
	if mustRequeue {
		log.Info("No need to generate new token, renewal is scheduled", "after", requeueAfter)
		return reconcile.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
	}

	log.Info("Requesting new token")

	expirationSeconds := int64(defaultValidityDuration / time.Second)
	tokResp, err := r.GardenAuthClientset.WorkloadIdentities(wi.Namespace).CreateToken(ctx, wi.Name, &authenticationv1alpha1.TokenRequest{
		Spec: authenticationv1alpha1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error requesting token: %w", err)
	}

	renewDuration := r.renewDuration(tokResp.Status.ExpirationTimestamp.Time)
	now := r.Clock.Now().UTC()
	_, err = controllerutil.CreateOrUpdate(ctx, r.SeedClient, secret, func() error {
		secret.Annotations = map[string]string{
			"workloadidentity.authentication.gardener.cloud/name":                  wi.Name,
			"workloadidentity.authentication.gardener.cloud/namespace":             wi.Namespace,
			"workloadidentity.authentication.gardener.cloud/uid":                   string(wi.UID),
			"workloadidentity.authentication.gardener.cloud/renewed-at":            now.Format(time.RFC3339),
			"workloadidentity.authentication.gardener.cloud/token-renew-timestamp": now.Add(renewDuration).Format(time.RFC3339),
		}

		secret.Labels = map[string]string{
			"authentication.gardener.cloud/purpose": "workloadidentity",
		}
		secret.Data = map[string][]byte{
			"token": []byte(tokResp.Status.Token),
			// TODO add extra config to data
		}
		return nil
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error writing workload identity token secret: %w", err)
	}

	log.Info("Successfully requested workload identity token and scheduled renewal", "after", renewDuration)
	return reconcile.Result{Requeue: true, RequeueAfter: renewDuration}, nil
}

func (r *Reconciler) renewDuration(expirationTimestamp time.Time) time.Duration {
	validityDuration := expirationTimestamp.UTC().Sub(r.Clock.Now().UTC())
	// renew tokens after roughly half of the time has passed
	return r.JitterFunc(validityDuration*50/100, 0.10)
}

func (r *Reconciler) requeue(ctx context.Context, secret *corev1.Secret) (bool, time.Duration, error) {
	if _, tokenExists := secret.Data["token"]; !tokenExists {
		return false, 0, nil
	}

	// TODO Should we check this annotation or instead parse the token
	// and its exp claim?
	renewTimestamp := secret.Annotations["workloadidentity.authentication.gardener.cloud/token-renew-timestamp"]
	if len(renewTimestamp) == 0 {
		return false, 0, nil
	}

	renewTime, err := time.Parse(time.RFC3339, renewTimestamp)
	if err != nil {
		return false, 0, fmt.Errorf("could not parse renew timestamp: %w", err)
	}

	if r.Clock.Now().UTC().Before(renewTime.UTC()) {
		return true, renewTime.UTC().Sub(r.Clock.Now().UTC()), nil
	}

	return false, 0, nil
}