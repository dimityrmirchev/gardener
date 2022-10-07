// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package auditbackend

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/gardener/gardener/pkg/operation/botanist/component"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultInterval is the default interval for retry operations.
	DefaultInterval = 5 * time.Second
	// DefaultSevereThreshold is the default threshold until an error reported by another component is treated as 'severe'.
	DefaultSevereThreshold = 30 * time.Second
	// DefaultTimeout is the default timeout and defines how long Gardener should wait
	// for a successful reconciliation of an auditbackend resource.
	DefaultTimeout = 3 * time.Minute
)

// TimeNow returns the current time. Exposed for testing.
var TimeNow = time.Now

// Values contains the values used to create a AuditBackend CRD
type Values struct {
	// Namespace is the namespace of the Shoot in the Seed
	Namespace string
	// Name is the name of the AuditBackend extension. Commonly the Shoot's name.
	Name string
	// Type is the type of AuditBackend plugin/extension
	Type string
	// ProviderConfig contains the provider config for the AuditBackend extension.
	ProviderConfig *runtime.RawExtension
}

// New creates a new instance of DeployWaiter for an AuditBackend.
func New(
	log logr.Logger,
	client client.Client,
	values *Values,
	waitInterval time.Duration,
	waitSevereThreshold time.Duration,
	waitTimeout time.Duration,
) component.DeployMigrateWaiter {
	return &backend{
		client:              client,
		log:                 log,
		values:              values,
		waitInterval:        waitInterval,
		waitSevereThreshold: waitSevereThreshold,
		waitTimeout:         waitTimeout,

		backend: &extensionsv1alpha1.AuditBackend{
			ObjectMeta: metav1.ObjectMeta{
				Name:      values.Name,
				Namespace: values.Namespace,
			},
		},
	}
}

type backend struct {
	values              *Values
	log                 logr.Logger
	client              client.Client
	waitInterval        time.Duration
	waitSevereThreshold time.Duration
	waitTimeout         time.Duration

	backend *extensionsv1alpha1.AuditBackend
}

// Deploy uses the seed client to create or update the AuditBackend custom resource in the Shoot namespace in the Seed
func (b *backend) Deploy(ctx context.Context) error {
	_, err := b.deploy(ctx, v1beta1constants.GardenerOperationReconcile)
	return err
}

// Restore uses the seed client and the ShootState to create the AuditBackend custom resource in the Shoot namespace in the Seed and restore its state
func (b *backend) Restore(ctx context.Context, shootState *v1alpha1.ShootState) error {
	return extensions.RestoreExtensionWithDeployFunction(
		ctx,
		b.client,
		shootState,
		extensionsv1alpha1.AuditBackendResource,
		b.deploy,
	)
}

// Migrate migrates the AuditBackend custom resource
func (n *backend) Migrate(ctx context.Context) error {
	return extensions.MigrateExtensionObject(
		ctx,
		n.client,
		n.backend,
	)
}

// WaitMigrate waits until the AuditBackend custom resource has been successfully migrated.
func (n *backend) WaitMigrate(ctx context.Context) error {
	return extensions.WaitUntilExtensionObjectMigrated(
		ctx,
		n.client,
		n.backend,
		extensionsv1alpha1.AuditBackendResource,
		n.waitInterval,
		n.waitTimeout,
	)
}

// Destroy deletes the AuditBackend CRD
func (n *backend) Destroy(ctx context.Context) error {
	return extensions.DeleteExtensionObject(
		ctx,
		n.client,
		n.backend,
	)
}

// Wait waits until the AuditBackend CRD is ready (deployed or restored)
func (n *backend) Wait(ctx context.Context) error {
	return extensions.WaitUntilExtensionObjectReady(
		ctx,
		n.client,
		n.log,
		n.backend,
		extensionsv1alpha1.AuditBackendResource,
		n.waitInterval,
		n.waitSevereThreshold,
		n.waitTimeout,
		nil,
	)
}

// WaitCleanup waits until the AuditBackend CRD is deleted
func (n *backend) WaitCleanup(ctx context.Context) error {
	return extensions.WaitUntilExtensionObjectDeleted(
		ctx,
		n.client,
		n.log,
		n.backend,
		extensionsv1alpha1.AuditBackendResource,
		n.waitInterval,
		n.waitTimeout,
	)
}

func (n *backend) deploy(ctx context.Context, operation string) (extensionsv1alpha1.Object, error) {
	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, n.client, n.backend, func() error {
		metav1.SetMetaDataAnnotation(&n.backend.ObjectMeta, v1beta1constants.GardenerOperation, operation)
		metav1.SetMetaDataAnnotation(&n.backend.ObjectMeta, v1beta1constants.GardenerTimestamp, TimeNow().UTC().String())

		n.backend.Spec = extensionsv1alpha1.AuditBackendSpec{
			DefaultSpec: extensionsv1alpha1.DefaultSpec{
				Type:           n.values.Type,
				ProviderConfig: n.values.ProviderConfig,
			},
		}

		return nil
	})

	return n.backend, err
}
