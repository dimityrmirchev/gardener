// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"context"
	"fmt"
	"strings"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/features"
	gardenletfeatures "github.com/gardener/gardener/pkg/gardenlet/features"
	"github.com/gardener/gardener/pkg/operation/botanist/component"
	"github.com/gardener/gardener/pkg/operation/botanist/component/namespaces"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploySeedNamespace creates a namespace in the Seed cluster which is used to deploy all the control plane
// components for the Shoot cluster. Moreover, the cloud provider configuration and all the secrets will be
// stored as ConfigMaps/Secrets.
func (b *Botanist) DeploySeedNamespace(ctx context.Context) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Shoot.SeedNamespace,
		},
	}

	if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, b.SeedClientSet.Client(), namespace, func() error {
		requiredExtensions, err := b.getShootRequiredExtensionTypes(ctx)
		if err != nil {
			return err
		}

		metav1.SetMetaDataAnnotation(&namespace.ObjectMeta, v1beta1constants.ShootUID, string(b.Shoot.GetInfo().Status.UID))
		metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.GardenRole, v1beta1constants.GardenRoleShoot)
		metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelSeedProvider, b.Seed.GetInfo().Spec.Provider.Type)
		metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelShootProvider, b.Shoot.GetInfo().Spec.Provider.Type)
		metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelNetworkingProvider, b.Shoot.GetInfo().Spec.Networking.Type)

		delete(namespace.Labels, v1beta1constants.LabelAuditBackendProvider)
		if b.Shoot.IsAuditBackendEnabled() {
			metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelAuditBackendProvider, b.Shoot.GetInfo().Spec.Kubernetes.KubeAPIServer.AuditConfig.Backend.Type)
		}

		// Remove all old extension labels before reconciling the new extension labels.
		for k := range namespace.Labels {
			if strings.HasPrefix(k, v1beta1constants.LabelExtensionPrefix) {
				delete(namespace.Labels, k)
			}
		}
		for extensionType := range requiredExtensions {
			metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelExtensionPrefix+extensionType, "true")
		}

		metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelBackupProvider, b.Seed.GetInfo().Spec.Provider.Type)
		if b.Seed.GetInfo().Spec.Backup != nil {
			metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.LabelBackupProvider, b.Seed.GetInfo().Spec.Backup.Provider)
		}

		// Label namespace to pin all control-plane pods of a shoot cluster to one zone
		// if the seed has workers across different availability zones.
		zone := namespace.Labels[v1beta1constants.ShootControlPlaneEnforceZone]
		delete(namespace.Labels, v1beta1constants.ShootControlPlaneEnforceZone)
		if zonePinningRequired(b.Shoot.GetInfo(), b.Seed.GetInfo()) {
			metav1.SetMetaDataLabel(&namespace.ObjectMeta, v1beta1constants.ShootControlPlaneEnforceZone, zone)
		}

		return nil
	}); err != nil {
		return err
	}

	b.SeedNamespaceObject = namespace
	return nil
}

func zonePinningRequired(shoot *gardencorev1beta1.Shoot, seed *gardencorev1beta1.Seed) bool {
	if !gardenletfeatures.FeatureGate.Enabled(features.HAControlPlanes) {
		return false
	}

	if !helper.IsMultiZonalSeed(seed) {
		return false
	}

	failureToleranceType := helper.GetFailureToleranceType(shoot)
	return failureToleranceType == nil || helper.IsFailureToleranceTypeNode(failureToleranceType)
}

// AddZoneInformationToSeedNamespace adds the name of the availability zone
// in which pods of a non-HA or single-zonal shoot run to the zone-pinning annotation.
func (b *Botanist) AddZoneInformationToSeedNamespace(ctx context.Context) error {
	if !zonePinningRequired(b.Shoot.GetInfo(), b.Seed.GetInfo()) {
		return nil
	}

	// Let's assume we can take any pod from the list to extract the zone information because they are all scheduled with
	// a zone affinity added by the pod-zone-affinity webhook of GRM.
	pods := &corev1.PodList{}
	if err := b.SeedClientSet.Client().List(ctx, pods, client.InNamespace(b.Shoot.SeedNamespace)); err != nil {
		return nil
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("zone information cannot be extracted because no running pods found in control-plane")
	}

	var nodeName string
	for _, pod := range pods.Items {
		if pod.Spec.NodeName == "" {
			continue
		}
		nodeName = pod.Spec.NodeName
		break
	}

	if nodeName == "" {
		return fmt.Errorf("zone information cannot be extracted because no pods have been scheduled yet")
	}

	node := &corev1.Node{}
	if err := b.SeedClientSet.Client().Get(ctx, kutil.Key(nodeName), node); err != nil {
		return fmt.Errorf("zone information cannot be extracted: %w", err)
	}

	zone := node.Labels[corev1.LabelTopologyZone]
	if zone == "" {
		return fmt.Errorf("zone information cannot be extracted because node %q does not contain any zone information", node.Name)
	}

	patch := client.MergeFrom(b.SeedNamespaceObject.DeepCopy())
	metav1.SetMetaDataLabel(&b.SeedNamespaceObject.ObjectMeta, v1beta1constants.ShootControlPlaneEnforceZone, zone)

	if err := b.SeedClientSet.Client().Patch(ctx, b.SeedNamespaceObject, patch); err != nil {
		return err
	}

	return nil
}

// DeleteSeedNamespace deletes the namespace in the Seed cluster which holds the control plane components. The built-in
// garbage collection in Kubernetes will automatically delete all resources which belong to this namespace. This
// comprises volumes and load balancers as well.
func (b *Botanist) DeleteSeedNamespace(ctx context.Context) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Shoot.SeedNamespace,
		},
	}

	err := b.SeedClientSet.Client().Delete(ctx, namespace, kubernetes.DefaultDeleteOptions...)
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		return nil
	}

	return err
}

// WaitUntilSeedNamespaceDeleted waits until the namespace of the Shoot cluster within the Seed cluster is deleted.
func (b *Botanist) WaitUntilSeedNamespaceDeleted(ctx context.Context) error {
	return retry.UntilTimeout(ctx, 5*time.Second, 900*time.Second, func(ctx context.Context) (done bool, err error) {
		if err := b.SeedClientSet.Client().Get(ctx, client.ObjectKey{Name: b.Shoot.SeedNamespace}, &corev1.Namespace{}); err != nil {
			if apierrors.IsNotFound(err) {
				return retry.Ok()
			}
			return retry.SevereError(err)
		}
		b.Logger.Info("Waiting until the namespace has been cleaned up and deleted in the Seed cluster", "namespaceName", b.Shoot.SeedNamespace)
		return retry.MinorError(fmt.Errorf("namespace %q is not yet cleaned up", b.Shoot.SeedNamespace))
	})
}

// DefaultShootNamespaces returns a deployer for the shoot namespaces.
func (b *Botanist) DefaultShootNamespaces() component.DeployWaiter {
	return namespaces.New(b.SeedClientSet.Client(), b.Shoot.SeedNamespace)
}

// getShootRequiredExtensionTypes returns all extension types that are enabled or explicitly disabled for the shoot.
// The function considers only extensions of kind `Extension`.
func (b *Botanist) getShootRequiredExtensionTypes(ctx context.Context) (sets.String, error) {
	controllerRegistrationList := &gardencorev1beta1.ControllerRegistrationList{}
	if err := b.GardenClient.List(ctx, controllerRegistrationList); err != nil {
		return nil, err
	}

	types := sets.String{}
	for _, reg := range controllerRegistrationList.Items {
		for _, res := range reg.Spec.Resources {
			if res.Kind == extensionsv1alpha1.ExtensionResource && pointer.BoolDeref(res.GloballyEnabled, false) {
				types.Insert(res.Type)
			}
		}
	}

	for _, extension := range b.Shoot.GetInfo().Spec.Extensions {
		if pointer.BoolDeref(extension.Disabled, false) {
			types.Delete(extension.Type)
		} else {
			types.Insert(extension.Type)
		}
	}

	return types, nil
}
