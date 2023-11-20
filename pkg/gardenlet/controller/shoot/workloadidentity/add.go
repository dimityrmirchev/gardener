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
	"fmt"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
		WithOptions(controller.Options{MaxConcurrentReconciles: 20}). // TODO make this configurable
		WatchesRawSource(
			source.Kind(gardenCluster.GetCache(), &gardencorev1beta1.Shoot{}),
			&handler.EnqueueRequestForObject{},
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
