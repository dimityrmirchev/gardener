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

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1clientset "github.com/gardener/gardener/pkg/client/authentication/clientset/versioned/typed/authentication/v1alpha1"
	predicateutils "github.com/gardener/gardener/pkg/controllerutils/predicate"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
)

// ControllerName is the name of this controller.
const ControllerName = "workload-identity-shoot"

// AddToManager adds Reconciler to the given manager.
func (r *Reconciler) AddToManager(mgr manager.Manager, gardenCluster, seedCluster cluster.Cluster) error {
	if r.GardenClient == nil {
		r.GardenClient = gardenCluster.GetClient()
	}
	if r.GardenAuthClientset == nil {
		var err error
		r.GardenAuthClientset, err = authenticationv1alpha1clientset.NewForConfig(gardenCluster.GetConfig())
		if err != nil {
			return fmt.Errorf("could not create authenticationv1alpha1Client: %w", err)
		}
	}

	if r.SeedClient == nil {
		r.SeedClient = seedCluster.GetClient()
	}

	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}

	if r.JitterFunc == nil {
		r.JitterFunc = wait.Jitter
	}

	return builder.
		ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			// TODO make this configurable
			MaxConcurrentReconciles: 20,
			RateLimiter:             workqueue.NewWithMaxWaitRateLimiter(workqueue.DefaultControllerRateLimiter(), time.Minute),
		}).
		WatchesRawSource(
			source.Kind(gardenCluster.GetCache(), &gardencorev1beta1.Shoot{}),
			eventHandler(),
			builder.WithPredicates(
				predicateutils.SeedNamePredicate(r.SeedName, gardenerutils.GetShootSeedNames),
				predicate.Funcs{
					CreateFunc:  func(e event.CreateEvent) bool { return isRelevantShoot(e.Object) },
					UpdateFunc:  func(e event.UpdateEvent) bool { return isRelevantShootUpdate(e.ObjectOld, e.ObjectNew) },
					DeleteFunc:  func(e event.DeleteEvent) bool { return isRelevantShoot(e.Object) },
					GenericFunc: func(e event.GenericEvent) bool { return false },
				},
			),
		).
		Complete(r)
}

func isRelevantShoot(obj client.Object) bool {
	shoot, ok := obj.(*gardencorev1beta1.Shoot)
	if !ok {
		return false
	}
	return shoot.Labels != nil && shoot.Labels["workloadidentity"] != ""
}

func isRelevantShootUpdate(old, new client.Object) bool {
	return isRelevantShoot(old) || isRelevantShoot(new)
}

// eventHandler returns an event handler.
func eventHandler() handler.EventHandler {
	return &handler.Funcs{
		CreateFunc: func(_ context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
			_, ok := e.Object.(*gardencorev1beta1.Shoot)
			if !ok {
				return
			}

			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      e.Object.GetName(),
				Namespace: e.Object.GetNamespace(),
			}})
		},
		UpdateFunc: func(_ context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			new, ok := e.ObjectNew.(*gardencorev1beta1.Shoot)
			if !ok {
				return
			}

			old, ok := e.ObjectOld.(*gardencorev1beta1.Shoot)
			if !ok {
				return
			}

			// TODO maybe revisit this
			// for now lets requeue if the spec changes
			if apiequality.Semantic.DeepEqual(old.Spec, new.Spec) {
				return
			}

			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      e.ObjectNew.GetName(),
				Namespace: e.ObjectNew.GetNamespace(),
			}})
		},
		DeleteFunc: func(_ context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
			_, ok := e.Object.(*gardencorev1beta1.Shoot)
			if !ok {
				return
			}

			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      e.Object.GetName(),
				Namespace: e.Object.GetNamespace(),
			}})
		},
	}
}
