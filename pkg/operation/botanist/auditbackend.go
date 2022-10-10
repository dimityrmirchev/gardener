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

package botanist

import (
	"context"

	"github.com/gardener/gardener/pkg/operation/botanist/component"
	"github.com/gardener/gardener/pkg/operation/botanist/component/extensions/auditbackend"
)

// DefaultAuditBackend creates the default deployer for the AuditBackend custom resource.
func (b *Botanist) DefaultAuditBackend() component.DeployMigrateWaiter {
	values := &auditbackend.Values{
		Namespace: b.Shoot.SeedNamespace,
		Name:      b.Shoot.GetInfo().Name,
	}
	if b.Shoot.IsAuditBackendEnabled() {
		values.Type = b.Shoot.GetInfo().Spec.Kubernetes.KubeAPIServer.AuditConfig.Backend.Type
		values.ProviderConfig = b.Shoot.GetInfo().Spec.Kubernetes.KubeAPIServer.AuditConfig.Backend.ProviderConfig
	}

	return auditbackend.New(
		b.Logger,
		b.SeedClientSet.Client(),
		values,
		auditbackend.DefaultInterval,
		auditbackend.DefaultSevereThreshold,
		auditbackend.DefaultTimeout,
	)
}

// DeployAuditBackend deploys the AuditBackend custom resource and triggers the restore operation in case
// the Shoot is in the restore phase of the control plane migration.
func (b *Botanist) DeployAuditBackend(ctx context.Context) error {
	if b.isRestorePhase() {
		return b.Shoot.Components.Extensions.AuditBackend.Restore(ctx, b.GetShootState())
	}

	return b.Shoot.Components.Extensions.AuditBackend.Deploy(ctx)
}

// DestroyAuditBackend destroys the AuditBackend custom resource.
func (b *Botanist) DestroyAuditBackend(ctx context.Context) error {
	if b.isRestorePhase() {
		return b.Shoot.Components.Extensions.AuditBackend.Restore(ctx, b.GetShootState())
	}

	return b.Shoot.Components.Extensions.AuditBackend.Deploy(ctx)
}
