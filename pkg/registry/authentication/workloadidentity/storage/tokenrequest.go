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

package storage

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"

	authenticationapi "github.com/gardener/gardener/pkg/apis/authentication"
	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
)

// TokenRequestREST implements a RESTStorage for workloadidentities/token.
type TokenRequestREST struct {
	maxExpirationSeconds int64
}

var (
	_ = rest.NamedCreater(&TokenRequestREST{})
	_ = rest.GroupVersionKindProvider(&TokenRequestREST{})

	gvk = schema.GroupVersionKind{
		Group:   authenticationv1alpha1.SchemeGroupVersion.Group,
		Version: authenticationv1alpha1.SchemeGroupVersion.Version,
		Kind:    "TokenRequest",
	}
)

// New returns and instance of TokenRequest
func (r *TokenRequestREST) New() runtime.Object {
	return &authenticationapi.TokenRequest{}
}

// Destroy cleans up its resources on shutdown.
func (r *TokenRequestREST) Destroy() {
	// Given that underlying store is shared with REST,
	// we don't destroy it here explicitly.
}

// Create returns a TokenRequest with a token based on
// the audiences of the workload identity.
func (r *TokenRequestREST) Create(ctx context.Context, name string, obj runtime.Object, createValidation rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj.DeepCopyObject()); err != nil {
			return nil, err
		}
	}
	out := obj.(*authenticationapi.TokenRequest)
	// TODO: implement this!
	out.Status.Token = "test"
	return out, nil
}

// GroupVersionKind returns authentication.gardener.cloud/v1alpha1 for TokenRequest.
func (r *TokenRequestREST) GroupVersionKind(schema.GroupVersion) schema.GroupVersionKind {
	return gvk
}
