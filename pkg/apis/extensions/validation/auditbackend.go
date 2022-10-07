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

package validation

import (
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateAuditBackend validates an AuditBackend object.
func ValidateAuditBackend(auditBackend *extensionsv1alpha1.AuditBackend) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&auditBackend.ObjectMeta, true, apivalidation.NameIsDNSSubdomain, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateAuditBackendSpec(&auditBackend.Spec, field.NewPath("spec"))...)

	return allErrs
}

// ValidateAuditBackendUpdate validates an AuditBackend object before an update.
func ValidateAuditBackendUpdate(new, old *extensionsv1alpha1.AuditBackend) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&new.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateAuditBackendSpecUpdate(&new.Spec, &old.Spec, new.DeletionTimestamp != nil, field.NewPath("spec"))...)
	allErrs = append(allErrs, ValidateAuditBackend(new)...)

	return allErrs
}

// ValidateAuditBackendSpec validates the specification of an AuditBackend object.
func ValidateAuditBackendSpec(spec *extensionsv1alpha1.AuditBackendSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(spec.Type) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "field is required"))
	}

	var cidrs []cidrvalidation.CIDR

	allErrs = append(allErrs, cidrvalidation.ValidateCIDRParse(cidrs...)...)
	allErrs = append(allErrs, cidrvalidation.ValidateCIDROverlap(cidrs, false)...)

	return allErrs
}

// ValidateAuditBackendSpecUpdate validates the spec of an AuditBackend object before an update.
func ValidateAuditBackendSpecUpdate(new, old *extensionsv1alpha1.AuditBackendSpec, deletionTimestampSet bool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if deletionTimestampSet && !apiequality.Semantic.DeepEqual(new, old) {
		allErrs = append(allErrs, apivalidation.ValidateImmutableField(new, old, fldPath)...)
		return allErrs
	}

	allErrs = append(allErrs, apivalidation.ValidateImmutableField(new.Type, old.Type, fldPath.Child("type"))...)

	return allErrs
}

// ValidateAuditBackendStatus validates the status of an AuditBackend object.
func ValidateAuditBackendStatus(status *extensionsv1alpha1.AuditBackendStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	return allErrs
}

// ValidateAuditBackendStatusUpdate validates the status field of an AuditBackend object before an update.
func ValidateAuditBackendStatusUpdate(newStatus, oldStatus *extensionsv1alpha1.AuditBackendStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	return allErrs
}
