/*
Copyright SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	"context"
	time "time"

	authentication "github.com/gardener/gardener/pkg/apis/authentication"
	clientsetinternalversion "github.com/gardener/gardener/pkg/client/authentication/clientset/internalversion"
	internalinterfaces "github.com/gardener/gardener/pkg/client/authentication/informers/internalversion/internalinterfaces"
	internalversion "github.com/gardener/gardener/pkg/client/authentication/listers/authentication/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// WorkloadIdentityInformer provides access to a shared informer and lister for
// WorkloadIdentities.
type WorkloadIdentityInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.WorkloadIdentityLister
}

type workloadIdentityInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewWorkloadIdentityInformer constructs a new informer for WorkloadIdentity type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewWorkloadIdentityInformer(client clientsetinternalversion.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredWorkloadIdentityInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredWorkloadIdentityInformer constructs a new informer for WorkloadIdentity type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredWorkloadIdentityInformer(client clientsetinternalversion.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Authentication().WorkloadIdentities(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Authentication().WorkloadIdentities(namespace).Watch(context.TODO(), options)
			},
		},
		&authentication.WorkloadIdentity{},
		resyncPeriod,
		indexers,
	)
}

func (f *workloadIdentityInformer) defaultInformer(client clientsetinternalversion.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredWorkloadIdentityInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *workloadIdentityInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&authentication.WorkloadIdentity{}, f.defaultInformer)
}

func (f *workloadIdentityInformer) Lister() internalversion.WorkloadIdentityLister {
	return internalversion.NewWorkloadIdentityLister(f.Informer().GetIndexer())
}
