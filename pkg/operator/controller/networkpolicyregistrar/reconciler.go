// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package networkpolicyregistrar

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gardencore "github.com/gardener/gardener/pkg/apis/core"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/gardener/gardener/pkg/controller/networkpolicy"
	"github.com/gardener/gardener/pkg/operator/apis/config"
)

// Reconciler adds the NetworkPolicy controller to the manager.
type Reconciler struct {
	Manager manager.Manager
	Config  config.NetworkPolicyControllerConfiguration

	added bool
}

// Reconcile performs the main reconciliation logic.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	if r.added {
		return reconcile.Result{}, nil
	}

	garden := &operatorv1alpha1.Garden{}
	if err := r.Manager.GetClient().Get(ctx, request.NamespacedName, garden); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if err := (&networkpolicy.Reconciler{
		ConcurrentSyncs:              r.Config.ConcurrentSyncs,
		AdditionalNamespaceSelectors: r.Config.AdditionalNamespaceSelectors,
		RuntimeNetworks: networkpolicy.RuntimeNetworkConfig{
			// gardener-operator only supports IPv4 single-stack networking in the runtime cluster for now.
			IPFamilies: []gardencore.IPFamily{gardencore.IPFamilyIPv4},
			Nodes:      garden.Spec.RuntimeCluster.Networking.Nodes,
			Pods:       garden.Spec.RuntimeCluster.Networking.Pods,
			Services:   garden.Spec.RuntimeCluster.Networking.Services,
			BlockCIDRs: garden.Spec.RuntimeCluster.Networking.BlockCIDRs,
		},
	}).AddToManager(ctx, r.Manager, r.Manager); err != nil {
		return reconcile.Result{}, err
	}

	r.added = true
	return reconcile.Result{}, nil
}
