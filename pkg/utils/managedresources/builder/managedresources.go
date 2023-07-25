// Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package builder

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
)

// ManagedResource is a structure managing a ManagedResource.
type ManagedResource struct {
	client   client.Client
	resource *resourcesv1alpha1.ManagedResource

	labels, annotations map[string]string
}

// NewManagedResource creates a new builder for a ManagedResource.
func NewManagedResource(client client.Client) *ManagedResource {
	return &ManagedResource{
		client:   client,
		resource: &resourcesv1alpha1.ManagedResource{},
	}
}

// WithNamespacedName sets the namespace and name.
func (m *ManagedResource) WithNamespacedName(namespace, name string) *ManagedResource {
	m.resource.Namespace = namespace
	m.resource.Name = name
	return m
}

// WithLabels sets the labels.
func (m *ManagedResource) WithLabels(labels map[string]string) *ManagedResource {
	m.labels = labels
	return m
}

// WithAnnotations sets the annotations.
func (m *ManagedResource) WithAnnotations(annotations map[string]string) *ManagedResource {
	m.annotations = annotations
	return m
}

// WithClass sets the Class field.
func (m *ManagedResource) WithClass(name string) *ManagedResource {
	if name == "" {
		m.resource.Spec.Class = nil
	} else {
		m.resource.Spec.Class = &name
	}
	return m
}

// WithSecretRef adds a reference with the given name to the SecretRefs field.
func (m *ManagedResource) WithSecretRef(secretRefName string) *ManagedResource {
	m.resource.Spec.SecretRefs = append(m.resource.Spec.SecretRefs, corev1.LocalObjectReference{Name: secretRefName})
	return m
}

// WithSecretRefs sets the SecretRefs field.
func (m *ManagedResource) WithSecretRefs(secretRefs []corev1.LocalObjectReference) *ManagedResource {
	m.resource.Spec.SecretRefs = append(m.resource.Spec.SecretRefs, secretRefs...)
	return m
}

// WithInjectedLabels sets the InjectLabels field.
func (m *ManagedResource) WithInjectedLabels(labelsToInject map[string]string) *ManagedResource {
	m.resource.Spec.InjectLabels = labelsToInject
	return m
}

// ForceOverwriteAnnotations sets the ForceOverwriteAnnotations field.
func (m *ManagedResource) ForceOverwriteAnnotations(v bool) *ManagedResource {
	m.resource.Spec.ForceOverwriteAnnotations = &v
	return m
}

// ForceOverwriteLabels sets the ForceOverwriteLabels field.
func (m *ManagedResource) ForceOverwriteLabels(v bool) *ManagedResource {
	m.resource.Spec.ForceOverwriteLabels = &v
	return m
}

// KeepObjects sets the KeepObjects field.
func (m *ManagedResource) KeepObjects(v bool) *ManagedResource {
	m.resource.Spec.KeepObjects = &v
	return m
}

// DeletePersistentVolumeClaims sets the DeletePersistentVolumeClaims field.
func (m *ManagedResource) DeletePersistentVolumeClaims(v bool) *ManagedResource {
	m.resource.Spec.DeletePersistentVolumeClaims = &v
	return m
}

// Reconcile creates or updates the ManagedResource as well as marks all referenced secrets as garbage collectable.
func (m *ManagedResource) Reconcile(ctx context.Context) error {
	resource := &resourcesv1alpha1.ManagedResource{
		ObjectMeta: metav1.ObjectMeta{Name: m.resource.Name, Namespace: m.resource.Namespace},
	}

	mutateFn := func(obj *resourcesv1alpha1.ManagedResource) {
		for k, v := range m.labels {
			metav1.SetMetaDataLabel(&obj.ObjectMeta, k, v)
		}

		for k, v := range m.annotations {
			metav1.SetMetaDataAnnotation(&obj.ObjectMeta, k, v)
		}

		obj.Spec = m.resource.Spec

		// the annotations should be injected after the spec is updated!
		utilruntime.Must(references.InjectAnnotations(obj))
	}

	if err := m.client.Get(ctx, client.ObjectKeyFromObject(resource), resource); err != nil {
		if apierrors.IsNotFound(err) {
			// if the mr is not found just create it
			mutateFn(resource)

			// only mark the new secrets
			if err := markSecretsAsGarbageCollectable(ctx, m.client, secretsFromRefs(resource, sets.New[string]())); err != nil {
				return err
			}

			return m.client.Create(ctx, resource)
		}
		return err
	}

	// mark all old secrets as garbage collectable
	// TODO(dimityrmirchev): Cleanup after Gardener v1.90
	excludedNames := sets.New[string]()
	oldSecrets := secretsFromRefs(resource, excludedNames)
	if err := markSecretsAsGarbageCollectable(ctx, m.client, oldSecrets); err != nil {
		return err
	}

	// Update the managed resource if necessary
	existing := resource.DeepCopyObject()
	mutateFn(resource)
	if equality.Semantic.DeepEqual(existing, resource) {
		return nil
	}

	// exclude all already patched secrets
	for _, s := range oldSecrets {
		excludedNames.Insert(s.Name)
	}

	// mark new secrets (if any) as garbage collectable
	newSecrets := secretsFromRefs(resource, excludedNames)
	if err := markSecretsAsGarbageCollectable(ctx, m.client, newSecrets); err != nil {
		return err
	}

	return m.client.Update(ctx, resource)
}

// Delete deletes the ManagedResource.
func (m *ManagedResource) Delete(ctx context.Context) error {
	return client.IgnoreNotFound(m.client.Delete(ctx, m.resource))
}

func markSecretsAsGarbageCollectable(ctx context.Context, c client.Client, secrets []*corev1.Secret) error {
	for _, secret := range secrets {
		if err := c.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
			return err
		}
		patch := client.StrategicMergeFrom(secret.DeepCopy())
		metav1.SetMetaDataLabel(&secret.ObjectMeta, references.LabelKeyGarbageCollectable, references.LabelValueGarbageCollectable)
		if err := c.Patch(ctx, secret, patch); err != nil {
			return err
		}
	}
	return nil
}

func secretsFromRefs(obj *resourcesv1alpha1.ManagedResource, excludedNames sets.Set[string]) []*corev1.Secret {
	secrets := make([]*corev1.Secret, 0, len(obj.Spec.SecretRefs))
	for _, secretRef := range obj.Spec.SecretRefs {
		if !excludedNames.Has(secretRef.Name) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretRef.Name,
					Namespace: obj.Namespace,
				},
			}
			secrets = append(secrets, secret)
		}
	}
	return secrets
}
