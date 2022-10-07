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

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-logr/logr"
)

// Actuator acts upon AuditBackend resources.
type Actuator interface {
	// Reconcile reconciles the AuditBackend resource.
	Reconcile(context.Context, logr.Logger, *extensionsv1alpha1.AuditBackend) error
	// Delete deletes the AuditBackend resource.
	Delete(context.Context, logr.Logger, *extensionsv1alpha1.AuditBackend) error
	// Restore restores the AuditBackend resource.
	Restore(context.Context, logr.Logger, *extensionsv1alpha1.AuditBackend) error
	// Migrate migrates the AuditBackend resource.
	Migrate(context.Context, logr.Logger, *extensionsv1alpha1.AuditBackend) error
}
