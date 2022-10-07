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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ Object = (*AuditBackend)(nil)

// AuditBackendResource is a constant for the name of the AuditBackend resource.
const AuditBackendResource = "AuditBackend"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Namespaced,path=auditbackends,singular=auditbackend
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Type,JSONPath=".spec.type",type=string,description="The type of the auditlog backend provider for this resource."
// +kubebuilder:printcolumn:name=Status,JSONPath=".status.lastOperation.state",type=string,description="Status of auditbackend resource."
// +kubebuilder:printcolumn:name=Age,JSONPath=".metadata.creationTimestamp",type=date,description="creation timestamp"

// AuditBackend is the specification for cluster auditlog service.
type AuditBackend struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the AuditBackend.
	// If the object's deletion timestamp is set, this field is immutable.
	Spec AuditBackendSpec `json:"spec"`
	// +optional
	Status AuditBackendStatus `json:"status"`
}

// GetExtensionSpec implements Object.
func (a *AuditBackend) GetExtensionSpec() Spec {
	return &a.Spec
}

// GetExtensionStatus implements Object.
func (a *AuditBackend) GetExtensionStatus() Status {
	return &a.Status
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuditBackendList is a list of AuditBackend resources.
type AuditBackendList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of AuditBackends.
	Items []AuditBackend `json:"items"`
}

// AuditBackendSpec is the spec for an AuditBackend resource.
type AuditBackendSpec struct {
	// DefaultSpec is a structure containing common fields used by all extension resources.
	DefaultSpec `json:",inline"`
}

// AuditBackendStatus is the status for an AuditBackend resource.
type AuditBackendStatus struct {
	// DefaultStatus is a structure containing common fields used by all extension resources.
	DefaultStatus `json:",inline"`
}

// GetExtensionType returns the type of this AuditBackend resource.
func (a *AuditBackend) GetExtensionType() string {
	return a.Spec.Type
}
