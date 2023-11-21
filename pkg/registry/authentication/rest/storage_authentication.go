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

package rest

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/kubernetes/pkg/serviceaccount"

	"github.com/gardener/gardener/pkg/api"
	"github.com/gardener/gardener/pkg/apis/authentication"
	gardenauthv1beta1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	workloadidentitystore "github.com/gardener/gardener/pkg/registry/authentication/workloadidentity/storage"
)

// StorageProvider contains configurations related to the authentication resources.
type StorageProvider struct {
	TokenSigner serviceaccount.TokenGenerator
}

// NewRESTStorage creates a new API group info object and registers the v1alpha1 authentication storage.
func (p StorageProvider) NewRESTStorage(restOptionsGetter generic.RESTOptionsGetter, tokenGenerator serviceaccount.TokenGenerator) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(authentication.GroupName, api.Scheme, metav1.ParameterCodec, api.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[gardenauthv1beta1.SchemeGroupVersion.Version] = p.v1alpha1Storage(restOptionsGetter, tokenGenerator)
	return apiGroupInfo
}

// GroupName returns the authentication group name.
func (p StorageProvider) GroupName() string {
	return authentication.GroupName
}

func (p StorageProvider) v1alpha1Storage(restOptionsGetter generic.RESTOptionsGetter, tokenGenerator serviceaccount.TokenGenerator) map[string]rest.Storage {
	storage := map[string]rest.Storage{}

	workloadIdentityStorage := workloadidentitystore.NewStorage(restOptionsGetter, tokenGenerator)
	storage["workloadidentities"] = workloadIdentityStorage.WorkloadIdentity
	storage["workloadidentities/status"] = workloadIdentityStorage.Status
	storage["workloadidentities/token"] = workloadIdentityStorage.TokenRequest

	return storage
}