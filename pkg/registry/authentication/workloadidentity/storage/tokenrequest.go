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
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2/jwt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/kubernetes/pkg/serviceaccount"

	authenticationapi "github.com/gardener/gardener/pkg/apis/authentication"
	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/authentication/validation"
)

// TokenRequestREST implements a RESTStorage for workloadidentities/token.
type TokenRequestREST struct {
	maxExpirationSeconds int64
	tokenGenerator       serviceaccount.TokenGenerator
	workloadIdentities   rest.Getter
}

func NewTokenRequestREST(tokenGenerator serviceaccount.TokenGenerator, wiGetter rest.Getter) *TokenRequestREST {
	return &TokenRequestREST{
		tokenGenerator:     tokenGenerator,
		workloadIdentities: wiGetter,
	}
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
	req := obj.(*authenticationapi.TokenRequest)

	// Get the namespace from the context (populated from the URL).
	namespace, ok := genericapirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.NewBadRequest("namespace is required")
	}

	// require name/namespace in the body to match URL if specified
	if len(req.Name) > 0 && req.Name != name {
		errs := field.ErrorList{field.Invalid(field.NewPath("metadata").Child("name"), req.Name, "must match the workload identity name if specified")}
		return nil, errors.NewInvalid(gvk.GroupKind(), name, errs)
	}
	if len(req.Namespace) > 0 && req.Namespace != namespace {
		errs := field.ErrorList{field.Invalid(field.NewPath("metadata").Child("namespace"), req.Namespace, "must match the workload identity namespace if specified")}
		return nil, errors.NewInvalid(gvk.GroupKind(), name, errs)
	}

	// Lookup workload identity
	wiObj, err := r.workloadIdentities.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	wi := wiObj.(*authenticationapi.WorkloadIdentity)

	// Populate metadata fields if not set
	if len(req.Name) == 0 {
		req.Name = wi.Name
	}
	if len(req.Namespace) == 0 {
		req.Namespace = wi.Namespace
	}

	// Save current time before building the token, to make sure the expiration
	// returned in TokenRequestStatus would be <= the exp field in token.
	nowTime := time.Now()
	req.CreationTimestamp = metav1.NewTime(nowTime)
	req.Status = authenticationapi.TokenRequestStatus{}

	if errs := validation.ValidateTokenRequest(req); len(errs) != 0 {
		return nil, errors.NewInvalid(gvk.GroupKind(), name, errs)
	}

	if createValidation != nil {
		if err := createValidation(ctx, obj.DeepCopyObject()); err != nil {
			return nil, err
		}
	}

	var oneHourSec int64 = 3600
	if req.Spec.ExpirationSeconds > oneHourSec { // one hour
		req.Spec.ExpirationSeconds = oneHourSec
	}

	public, private := claims(*wi, req.Spec.ExpirationSeconds)
	token, err := r.tokenGenerator.GenerateToken(public, private)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	out := req.DeepCopy()
	out.Status = authenticationapi.TokenRequestStatus{
		Token:               token,
		ExpirationTimestamp: metav1.Time{Time: nowTime.Add(time.Duration(out.Spec.ExpirationSeconds) * time.Second)},
	}
	return out, nil
}

// GroupVersionKind returns authentication.gardener.cloud/v1alpha1 for TokenRequest.
func (r *TokenRequestREST) GroupVersionKind(schema.GroupVersion) schema.GroupVersionKind {
	return gvk
}

type privateClaims struct {
	Gardener gardener `json:"gardener.cloud,omitempty"`
}

type gardener struct {
	Namespace        string `json:"namespace,omitempty"`
	WorkloadIdentity ref    `json:"workloadidentity,omitempty"`
}

type ref struct {
	Name string `json:"name,omitempty"`
	UID  string `json:"uid,omitempty"`
}

func claims(wi authenticationapi.WorkloadIdentity, expirationSeconds int64) (*jwt.Claims, interface{}) {
	now := time.Now()
	sc := &jwt.Claims{
		Subject:   "gardener:workloadidentity:" + wi.Namespace + ":" + wi.Name,
		Audience:  jwt.Audience(wi.Spec.Audiences),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Expiry:    jwt.NewNumericDate(now.Add(time.Duration(expirationSeconds) * time.Second)),
	}
	pc := &privateClaims{
		Gardener: gardener{
			Namespace: wi.Namespace,
			WorkloadIdentity: ref{
				Name: wi.Name,
				UID:  string(wi.UID),
			},
		},
	}

	return sc, pc
}
