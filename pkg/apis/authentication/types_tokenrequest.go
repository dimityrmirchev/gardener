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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TokenRequest can be used to request a token with for a specific workload identity.
type TokenRequest struct {
	metav1.TypeMeta
	// Standard object metadata.
	metav1.ObjectMeta
	// Spec is the specification of the TokenRequest.
	Spec TokenRequestSpec
	// Status is the status of the TokenRequest.
	Status TokenRequestStatus
}

// TokenRequestStatus is the status of the TokenRequest containing
// the token.
type TokenRequestStatus struct {
	// Token is the bearer token.
	Token string
	// ExpirationTimestamp is the expiration timestamp of the returned credential.
	ExpirationTimestamp metav1.Time
}

// TokenRequestSpec contains the expiration time of the token.
type TokenRequestSpec struct {
	// ExpirationSeconds is the requested validity duration of the credential. The
	// credential issuer may return a credential with a different validity duration so a
	// client needs to check the 'expirationTimestamp' field in a response.
	// Defaults to 1 hour.
	ExpirationSeconds int64
}