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

// Package validation contains methods to validate kinds in the
// authentication.k8s.io API group.
package validation

import (
	"math"
	"time"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener/pkg/apis/authentication"
)

// ValidateWorkloadIdentity validates a WorkloadIdentity.
func ValidateWorkloadIdentity(wi *authentication.WorkloadIdentity) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")

	if len(wi.Spec.Audiences) == 0 {
		allErrs = append(allErrs, field.Invalid(specPath.Child("audiences"), wi.Spec.Audiences, "should specify at least one audience"))
	}

	return allErrs
}

// ValidateWorkloadIdentityUpdate validates a WorkloadIdentity object before an update.
func ValidateWorkloadIdentityUpdate(new, old *authentication.WorkloadIdentity) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&new.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateWorkloadIdentity(new)...)

	return allErrs
}

// ValidateWorkloadIdentityStatusUpdate validates the status field of a WorkloadIdentity object.
func ValidateWorkloadIdentityStatusUpdate(_, _ authentication.WorkloadIdentityStatus) field.ErrorList {
	allErrs := field.ErrorList{}

	return allErrs
}

// ValidateTokenRequest validates a TokenRequest.
func ValidateTokenRequest(tr *authentication.TokenRequest) field.ErrorList {
	allErrs := field.ErrorList{}
	specPath := field.NewPath("spec")

	const min = 10 * time.Minute
	if tr.Spec.ExpirationSeconds < int64(min.Seconds()) {
		allErrs = append(allErrs, field.Invalid(specPath.Child("expirationSeconds"), tr.Spec.ExpirationSeconds, "may not specify a duration less than 10 minutes"))
	}
	if tr.Spec.ExpirationSeconds > math.MaxUint32 {
		allErrs = append(allErrs, field.TooLong(specPath.Child("expirationSeconds"), tr.Spec.ExpirationSeconds, math.MaxUint32))
	}
	return allErrs
}
