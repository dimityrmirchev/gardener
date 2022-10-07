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

package validation_test

import (
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/gardener/gardener/pkg/apis/extensions/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("AuditBackend validation tests", func() {
	var auditBackend *extensionsv1alpha1.AuditBackend

	BeforeEach(func() {
		auditBackend = &extensionsv1alpha1.AuditBackend{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
			Spec: extensionsv1alpha1.AuditBackendSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           "provider",
					ProviderConfig: &runtime.RawExtension{},
				},
			},
		}
	})

	Describe("#ValidAuditBackend", func() {
		It("should forbid empty AuditBackend resources", func() {
			errorList := ValidateAuditBackend(&extensionsv1alpha1.AuditBackend{})

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.type"),
			}))))
		})

		It("should allow valid audit backend resources", func() {
			errorList := ValidateAuditBackend(auditBackend)

			Expect(errorList).To(BeEmpty())
		})
	})

	Describe("#ValidAuditBackendUpdate", func() {
		It("should prevent updating anything if deletion time stamp is set", func() {
			now := metav1.Now()
			auditBackend.DeletionTimestamp = &now

			newAuditBackend := prepareAuditBackendForUpdate(auditBackend)
			newAuditBackend.DeletionTimestamp = &now
			newAuditBackend.Spec.ProviderConfig = nil

			errorList := ValidateAuditBackendUpdate(newAuditBackend, auditBackend)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should prevent updating the type or the cidrs", func() {
			newAuditBackend := prepareAuditBackendForUpdate(auditBackend)
			newAuditBackend.Spec.Type = "changed-type"

			errorList := ValidateAuditBackendUpdate(newAuditBackend, auditBackend)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.type"),
			}))))
		})

		It("should allow updating the provider config", func() {
			newAuditBackend := prepareAuditBackendForUpdate(auditBackend)
			newAuditBackend.Spec.ProviderConfig = nil

			errorList := ValidateAuditBackendUpdate(newAuditBackend, auditBackend)

			Expect(errorList).To(BeEmpty())
		})
	})
})

func prepareAuditBackendForUpdate(obj *extensionsv1alpha1.AuditBackend) *extensionsv1alpha1.AuditBackend {
	newObj := obj.DeepCopy()
	newObj.ResourceVersion = "1"
	return newObj
}
