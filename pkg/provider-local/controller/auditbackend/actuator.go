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

package auditbackend

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/auditbackend"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/provider-local/local"
	"github.com/gardener/gardener/pkg/utils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplicationName string = "audit-backend"
)

type actuator struct {
	client client.Client
}

// NewActuator creates a new Actuator that updates the status of the handled AuditBackend resources.
func NewActuator() auditbackend.Actuator {
	return &actuator{}
}

func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, auditBackend *extensionsv1alpha1.AuditBackend) error {
	namespace := auditBackend.Namespace
	deployment := backendDeployment(namespace)
	service := backendService(namespace)

	for _, obj := range []client.Object{
		deployment,
		service,
	} {
		if err := a.client.Patch(ctx, obj, client.Apply, local.FieldOwner, client.ForceOwnership); err != nil {
			return err
		}
	}

	return nil

}

func (a *actuator) Delete(ctx context.Context, log logr.Logger, auditBackend *extensionsv1alpha1.AuditBackend) error {
	return kutil.DeleteObjects(ctx, a.client,
		emptyService(auditBackend.Namespace),
		emptyDeployment(auditBackend.Namespace),
	)
}

func (a *actuator) Migrate(ctx context.Context, log logr.Logger, auditBackend *extensionsv1alpha1.AuditBackend) error {
	return a.Delete(ctx, log, auditBackend)
}

func (a *actuator) Restore(ctx context.Context, log logr.Logger, auditBackend *extensionsv1alpha1.AuditBackend) error {
	return a.Reconcile(ctx, log, auditBackend)
}

func getLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name": ApplicationName,
	}
}

func emptyDeployment(namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ApplicationName,
		},
	}
}

func emptyService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ApplicationName,
		},
	}
}

func backendDeployment(namespace string) *appsv1.Deployment {
	var one int32 = 1
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ApplicationName,
			Labels:    getLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{MatchLabels: getLabels()},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: utils.MergeStringMaps(getLabels(), map[string]string{
						v1beta1constants.LabelNetworkPolicyToDNS:              v1beta1constants.LabelNetworkPolicyAllowed,
						v1beta1constants.LabelNetworkPolicyFromShootAPIServer: v1beta1constants.LabelNetworkPolicyAllowed,
						v1beta1constants.LabelNetworkPolicyFromPrometheus:     v1beta1constants.LabelNetworkPolicyAllowed,
					}),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            ApplicationName,
						Image:           "nginx:1.23.1",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{{
							Name:          "tcp",
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: 80,
						}},
					}},
				},
			},
		},
	}
}

func backendService(namespace string) *corev1.Service {
	port80 := intstr.FromInt(80)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: namespace,
			Labels:    getLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: getLabels(),
			Ports: []corev1.ServicePort{
				{
					Name:       "tcp",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: port80,
				},
			},
		},
	}
}
