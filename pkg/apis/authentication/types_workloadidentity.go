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

package authentication

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadIdentity holds certain properties related to Gardener managed workload communicating with external systems.
type WorkloadIdentity struct {
	metav1.TypeMeta
	// Standard object metadata.
	metav1.ObjectMeta
	// Spec defines the workload identity properties.
	Spec WorkloadIdentitySpec
	// Most recently observed status of the WorkloadIdentity.
	Status WorkloadIdentityStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadIdentityList is a collection of WorkloadIdentities.
type WorkloadIdentityList struct {
	metav1.TypeMeta
	// Standard list object metadata.
	metav1.ListMeta
	// Items is the list of WorkloadIdentities.
	Items []WorkloadIdentity
}

// WorkloadIdentitySpec is the specification of a WorkloadIdentity.
type WorkloadIdentitySpec struct {
	// Audiences represent the target systems which the current workload identity will be used against.
	Audiences []string
}

// WorkloadIdentityStatus holds the most recently observed status of the workload identity.
type WorkloadIdentityStatus struct {
	// ObservedGeneration is the most recent generation observed for this workload identity.
	ObservedGeneration int64
}
