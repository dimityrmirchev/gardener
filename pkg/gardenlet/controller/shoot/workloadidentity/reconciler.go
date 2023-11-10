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

// Reconciler requests and refreshes tokens via the Workload Identity TokenRequest API.
type Reconciler struct {
	GardenClient        client.Client
	SeedClient          client.Client
	GardenAuthClientset *authenticationv1alpha1clientset.AuthenticationV1alpha1Client
	Clock               clock.Clock
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

	log.Info("Requesting new token")

	wi := &authenticationv1alpha1.WorkloadIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shoot.Labels["workloadidentity"], // TODO fix this
			Namespace: req.NamespacedName.Namespace,
		},
	}

	wiKey := client.ObjectKeyFromObject(wi)
	if err := r.GardenClient.Get(ctx, wiKey, wi); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object is gone, stop reconciling", "workloadIdentity", wiKey.String())
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	tokResp, err := r.GardenAuthClientset.WorkloadIdentities(wi.Namespace).CreateToken(ctx, wi.Name, &authenticationv1alpha1.TokenRequest{
		Spec: authenticationv1alpha1.TokenRequestSpec{},
	}, metav1.CreateOptions{})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error requesting token: %w", err)
	}

	// TODO also use the uuid of a workload identity
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wi--" + wi.Name,
			Namespace: shoot.Status.TechnicalID,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.SeedClient, secret, func() error {
		secret.Data = map[string][]byte{
			"token": []byte(tokResp.Status.Token),
		}
		return nil
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error writing workload identity token secret: %w", err)
	}

	return reconcile.Result{RequeueAfter: time.Second * 600}, nil
}
