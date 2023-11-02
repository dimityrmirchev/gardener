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

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"

	"github.com/gardener/gardener/pkg/api"
	"github.com/gardener/gardener/pkg/apis/authentication"
	"github.com/gardener/gardener/pkg/apis/authentication/validation"
)

type workloadIdentityStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy defines the storage strategy for WorkloadIdentities.
var Strategy = workloadIdentityStrategy{api.Scheme, names.SimpleNameGenerator}

// Strategy should implement rest.RESTCreateUpdateStrategy
var _ rest.RESTCreateUpdateStrategy = workloadIdentityStrategy{}

func (workloadIdentityStrategy) NamespaceScoped() bool {
	return true
}

func (workloadIdentityStrategy) PrepareForCreate(_ context.Context, obj runtime.Object) {
	workloadIdentity := obj.(*authentication.WorkloadIdentity)

	workloadIdentity.Generation = 1
	workloadIdentity.Status = authentication.WorkloadIdentityStatus{}
}

func (workloadIdentityStrategy) PrepareForUpdate(_ context.Context, objNew, objOld runtime.Object) {
	new := objNew.(*authentication.WorkloadIdentity)
	old := objNew.(*authentication.WorkloadIdentity)

	new.Status = old.Status // can only be changed by workloadidentities/status subresource

	if mustIncreaseGeneration(old, new) {
		new.Generation = old.Generation + 1
	}
}

func mustIncreaseGeneration(old, new *authentication.WorkloadIdentity) bool {
	if !apiequality.Semantic.DeepEqual(old.Spec, new.Spec) {
		return true
	}

	// The deletion timestamp is set.
	if old.DeletionTimestamp == nil && new.DeletionTimestamp != nil {
		return true
	}

	return false
}

func (workloadIdentityStrategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	workloadIdentity := obj.(*authentication.WorkloadIdentity)
	return validation.ValidateWorkloadIdentity(workloadIdentity)
}

func (workloadIdentityStrategy) Canonicalize(_ runtime.Object) {
}

func (workloadIdentityStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (workloadIdentityStrategy) ValidateUpdate(_ context.Context, newObj, oldObj runtime.Object) field.ErrorList {
	new := newObj.(*authentication.WorkloadIdentity)
	old := oldObj.(*authentication.WorkloadIdentity)
	return validation.ValidateWorkloadIdentityUpdate(new, old)
}

func (workloadIdentityStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// WarningsOnCreate returns warnings to the client performing a create.
func (s workloadIdentityStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate returns warnings to the client performing the update.
func (s workloadIdentityStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type workloadIdentityStatusStrategy struct {
	workloadIdentityStrategy
}

// StatusStrategy defines the storage strategy for the status subresource of WorkloadIdentities.
var StatusStrategy = workloadIdentityStatusStrategy{Strategy}

func (workloadIdentityStatusStrategy) PrepareForUpdate(_ context.Context, newObj, oldObj runtime.Object) {
	new := newObj.(*authentication.WorkloadIdentity)
	old := oldObj.(*authentication.WorkloadIdentity)
	new.Spec = old.Spec
}

func (workloadIdentityStatusStrategy) ValidateUpdate(_ context.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateWorkloadIdentityStatusUpdate(obj.(*authentication.WorkloadIdentity).Status, old.(*authentication.WorkloadIdentity).Status)
}

func (workloadIdentityStatusStrategy) WarningsOnCreate(_ context.Context, _ runtime.Object) []string {
	return nil
}

func (workloadIdentityStatusStrategy) WarningsOnUpdate(_ context.Context, _, _ runtime.Object) []string {
	return nil
}
