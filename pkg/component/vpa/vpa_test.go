// Copyright 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package vpa_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	componenttest "github.com/gardener/gardener/pkg/component/test"
	. "github.com/gardener/gardener/pkg/component/vpa"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	retryfake "github.com/gardener/gardener/pkg/utils/retry/fake"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
)

var _ = Describe("VPA", func() {
	var (
		ctx = context.TODO()

		namespace    = "some-namespace"
		secretNameCA = "ca"

		genericTokenKubeconfigSecretName = "generic-token-kubeconfig"
		pathGenericKubeconfig            = "/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig/kubeconfig"

		runtimeKubernetesVersion = semver.MustParse("1.25.0")
		values                   = Values{
			SecretNameServerCA:       secretNameCA,
			RuntimeKubernetesVersion: runtimeKubernetesVersion,
		}

		c   client.Client
		sm  secretsmanager.Interface
		vpa component.DeployWaiter

		imageAdmissionController = "some-image:for-admission-controller"
		imageRecommender         = "some-image:for-recommender"
		imageUpdater             = "some-image:for-updater"

		livenessProbeVpa = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/health-check",
					Port:   intstr.IntOrString{Type: intstr.String, StrVal: "metrics"},
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 120,
			PeriodSeconds:       60,
			TimeoutSeconds:      30,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}

		valuesAdmissionController ValuesAdmissionController
		valuesRecommender         ValuesRecommender
		valuesUpdater             ValuesUpdater

		vpaUpdateModeAuto   = vpaautoscalingv1.UpdateModeAuto
		vpaControlledValues = vpaautoscalingv1.ContainerControlledValuesRequestsOnly
		maxUnavailable      = intstr.FromInt(1)

		webhookFailurePolicy      = admissionregistrationv1.Ignore
		webhookMatchPolicy        = admissionregistrationv1.Exact
		webhookReinvocationPolicy = admissionregistrationv1.NeverReinvocationPolicy
		webhookSideEffects        = admissionregistrationv1.SideEffectClassNone
		webhookScope              = admissionregistrationv1.AllScopes

		managedResourceName   string
		managedResource       *resourcesv1alpha1.ManagedResource
		managedResourceSecret *corev1.Secret

		serviceAccountUpdater     *corev1.ServiceAccount
		clusterRoleUpdater        *rbacv1.ClusterRole
		clusterRoleBindingUpdater *rbacv1.ClusterRoleBinding
		shootAccessSecretUpdater  *corev1.Secret
		deploymentUpdaterFor      func(bool, *metav1.Duration, *metav1.Duration, *int32, *float64, *float64, component.ClusterType) *appsv1.Deployment
		vpaUpdater                *vpaautoscalingv1.VerticalPodAutoscaler

		serviceAccountRecommender                    *corev1.ServiceAccount
		clusterRoleRecommenderMetricsReader          *rbacv1.ClusterRole
		clusterRoleBindingRecommenderMetricsReader   *rbacv1.ClusterRoleBinding
		clusterRoleRecommenderCheckpointActor        *rbacv1.ClusterRole
		clusterRoleBindingRecommenderCheckpointActor *rbacv1.ClusterRoleBinding
		serviceRecommenderFor                        func(component.ClusterType) *corev1.Service
		shootAccessSecretRecommender                 *corev1.Secret
		deploymentRecommenderFor                     func(bool, *metav1.Duration, *float64, component.ClusterType) *appsv1.Deployment
		vpaRecommender                               *vpaautoscalingv1.VerticalPodAutoscaler

		serviceAccountAdmissionController      *corev1.ServiceAccount
		clusterRoleAdmissionController         *rbacv1.ClusterRole
		clusterRoleBindingAdmissionController  *rbacv1.ClusterRoleBinding
		shootAccessSecretAdmissionController   *corev1.Secret
		serviceAdmissionControllerFor          func(component.ClusterType, bool) *corev1.Service
		deploymentAdmissionControllerFor       func(bool, component.ClusterType) *appsv1.Deployment
		podDisruptionBudgetAdmissionController *policyv1.PodDisruptionBudget
		vpaAdmissionController                 *vpaautoscalingv1.VerticalPodAutoscaler

		clusterRoleGeneralActor               *rbacv1.ClusterRole
		clusterRoleBindingGeneralActor        *rbacv1.ClusterRoleBinding
		clusterRoleGeneralTargetReader        *rbacv1.ClusterRole
		clusterRoleBindingGeneralTargetReader *rbacv1.ClusterRoleBinding
		mutatingWebhookConfiguration          *admissionregistrationv1.MutatingWebhookConfiguration
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		sm = fakesecretsmanager.New(c, namespace)

		valuesAdmissionController = ValuesAdmissionController{
			Image:             imageAdmissionController,
			PriorityClassName: "priority-admission-controller",
		}
		valuesRecommender = ValuesRecommender{
			Image:             imageRecommender,
			PriorityClassName: "priority-recommender",
		}
		valuesUpdater = ValuesUpdater{
			Image:             imageUpdater,
			PriorityClassName: "priority-updater",
		}

		vpa = New(c, namespace, sm, values)
		managedResourceName = ""

		By("Create secrets managed outside of this package for whose secretsmanager.Get() will be called")
		Expect(c.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca", Namespace: namespace}})).To(Succeed())
		Expect(c.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "generic-token-kubeconfig", Namespace: namespace}})).To(Succeed())

		serviceAccountUpdater = &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-updater",
				Namespace: namespace,
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		clusterRoleUpdater = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:evictioner",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apps", "extensions"},
					Resources: []string{"replicasets"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods/eviction"},
					Verbs:     []string{"create"},
				},
			},
		}
		clusterRoleBindingUpdater = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:evictioner",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
				Annotations: map[string]string{
					"resources.gardener.cloud/delete-on-invalid-update": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:vpa:target:evictioner",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "vpa-updater",
				Namespace: namespace,
			}},
		}
		shootAccessSecretUpdater = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-access-vpa-updater",
				Namespace: namespace,
				Labels: map[string]string{
					"resources.gardener.cloud/purpose": "token-requestor",
				},
				Annotations: map[string]string{
					"serviceaccount.resources.gardener.cloud/name":      "vpa-updater",
					"serviceaccount.resources.gardener.cloud/namespace": "kube-system",
				},
			},
			Type: corev1.SecretTypeOpaque,
		}
		deploymentUpdaterFor = func(
			withServiceAccount bool,
			interval *metav1.Duration,
			evictAfterOOMThreshold *metav1.Duration,
			evictionRateBurst *int32,
			evictionRateLimit *float64,
			evictionTolerance *float64,
			clusterType component.ClusterType,
		) *appsv1.Deployment {
			var (
				flagEvictionToleranceValue      = "0.500000"
				flagEvictionRateBurstValue      = "1"
				flagEvictionRateLimitValue      = "-1.000000"
				flagEvictAfterOomThresholdValue = "10m0s"
				flagUpdaterIntervalValue        = "1m0s"
			)

			if interval != nil {
				flagUpdaterIntervalValue = interval.Duration.String()
			}
			if evictAfterOOMThreshold != nil {
				flagEvictAfterOomThresholdValue = evictAfterOOMThreshold.Duration.String()
			}
			if evictionRateBurst != nil {
				flagEvictionRateBurstValue = fmt.Sprintf("%d", *evictionRateBurst)
			}
			if evictionRateLimit != nil {
				flagEvictionRateLimitValue = fmt.Sprintf("%f", *evictionRateLimit)
			}
			if evictionTolerance != nil {
				flagEvictionToleranceValue = fmt.Sprintf("%f", *evictionTolerance)
			}

			obj := &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vpa-updater",
					Namespace: namespace,
					Labels: map[string]string{
						"app":                 "vpa-updater",
						"gardener.cloud/role": "vpa",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas:             pointer.Int32(1),
					RevisionHistoryLimit: pointer.Int32(2),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "vpa-updater",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app":                              "vpa-updater",
								"gardener.cloud/role":              "vpa",
								"networking.gardener.cloud/to-dns": "allowed",
								"networking.resources.gardener.cloud/to-kube-apiserver-tcp-443": "allowed",
							},
						},
						Spec: corev1.PodSpec{
							PriorityClassName: valuesUpdater.PriorityClassName,
							Containers: []corev1.Container{{
								Name:            "updater",
								Image:           imageUpdater,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command:         []string{"./updater"},
								Args: []string{
									"--min-replicas=1",
									fmt.Sprintf("--eviction-tolerance=%s", flagEvictionToleranceValue),
									fmt.Sprintf("--eviction-rate-burst=%s", flagEvictionRateBurstValue),
									fmt.Sprintf("--eviction-rate-limit=%s", flagEvictionRateLimitValue),
									fmt.Sprintf("--evict-after-oom-threshold=%s", flagEvictAfterOomThresholdValue),
									fmt.Sprintf("--updater-interval=%s", flagUpdaterIntervalValue),
									"--stderrthreshold=info",
									"--v=2",
								},
								LivenessProbe: livenessProbeVpa,
								Ports: []corev1.ContainerPort{
									{
										Name:          "server",
										ContainerPort: 8080,
									},
									{
										Name:          "metrics",
										ContainerPort: 8943,
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("200Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("4Gi"),
									},
								},
							}},
						},
					},
				},
			}

			if withServiceAccount {
				obj.Spec.Template.Spec.ServiceAccountName = serviceAccountUpdater.Name
				obj.Spec.Template.Spec.Containers[0].Env = append(obj.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
					Name: "NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				})
			} else {
				obj.Labels["gardener.cloud/role"] = "controlplane"
				obj.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)
				obj.Spec.Template.Spec.Containers[0].Command = append(obj.Spec.Template.Spec.Containers[0].Command, "--kubeconfig="+pathGenericKubeconfig)

				Expect(gardenerutils.InjectGenericKubeconfig(obj, genericTokenKubeconfigSecretName, shootAccessSecretUpdater.Name)).To(Succeed())
			}

			return obj
		}
		vpaUpdater = &vpaautoscalingv1.VerticalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling.k8s.io/v1",
				Kind:       "VerticalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-updater",
				Namespace: namespace,
			},
			Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscalingv1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "vpa-updater",
				},
				UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{UpdateMode: &vpaUpdateModeAuto},
				ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
					ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
						{
							ContainerName:    "*",
							ControlledValues: &vpaControlledValues,
							MinAllowed: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("50Mi"),
							},
						},
					},
				},
			},
		}

		serviceAccountRecommender = &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-recommender",
				Namespace: namespace,
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		clusterRoleRecommenderMetricsReader = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:metrics-reader",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"metrics.k8s.io"},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list"},
				},
			},
		}
		clusterRoleBindingRecommenderMetricsReader = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:metrics-reader",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
				Annotations: map[string]string{
					"resources.gardener.cloud/delete-on-invalid-update": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:vpa:target:metrics-reader",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "vpa-recommender",
				Namespace: namespace,
			}},
		}
		clusterRoleRecommenderCheckpointActor = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:checkpoint-actor",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"poc.autoscaling.k8s.io"},
					Resources: []string{"verticalpodautoscalercheckpoints"},
					Verbs:     []string{"get", "list", "watch", "create", "patch", "delete"},
				},
				{
					APIGroups: []string{"autoscaling.k8s.io"},
					Resources: []string{"verticalpodautoscalercheckpoints"},
					Verbs:     []string{"get", "list", "watch", "create", "patch", "delete"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get", "list"},
				},
			},
		}
		clusterRoleBindingRecommenderCheckpointActor = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:checkpoint-actor",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
				Annotations: map[string]string{
					"resources.gardener.cloud/delete-on-invalid-update": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:vpa:target:checkpoint-actor",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "vpa-recommender",
				Namespace: namespace,
			}},
		}
		serviceRecommenderFor = func(clusterType component.ClusterType) *corev1.Service {
			obj := &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vpa-recommender",
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{"app": "vpa-recommender"},
					Ports: []corev1.ServicePort{{
						Port:       8942,
						TargetPort: intstr.FromInt(8942),
					}},
				},
			}

			if clusterType == "seed" {
				obj.Annotations = map[string]string{
					"networking.resources.gardener.cloud/from-all-seed-scrape-targets-allowed-ports": `[{"protocol":"TCP","port":8942}]`,
				}
			} else if clusterType == "shoot" {
				obj.Annotations = map[string]string{
					"networking.resources.gardener.cloud/namespace-selectors":                `[{"matchLabels":{"kubernetes.io/metadata.name":"garden"}}]`,
					"networking.resources.gardener.cloud/pod-label-selector-namespace-alias": "all-shoots",
				}
			}

			return obj
		}
		shootAccessSecretRecommender = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-access-vpa-recommender",
				Namespace: namespace,
				Labels: map[string]string{
					"resources.gardener.cloud/purpose": "token-requestor",
				},
				Annotations: map[string]string{
					"serviceaccount.resources.gardener.cloud/name":      "vpa-recommender",
					"serviceaccount.resources.gardener.cloud/namespace": "kube-system",
				},
			},
			Type: corev1.SecretTypeOpaque,
		}
		deploymentRecommenderFor = func(
			withServiceAccount bool,
			interval *metav1.Duration,
			recommendationMarginFraction *float64,
			clusterType component.ClusterType,
		) *appsv1.Deployment {
			var (
				flagRecommendationMarginFraction = "0.150000"
				flagRecommenderIntervalValue     = "1m0s"
			)

			if interval != nil {
				flagRecommenderIntervalValue = interval.Duration.String()
			}
			if recommendationMarginFraction != nil {
				flagRecommendationMarginFraction = fmt.Sprintf("%f", *recommendationMarginFraction)
			}

			var cpuRequest string
			var memoryRequest string
			if clusterType == component.ClusterTypeShoot {
				cpuRequest = "30m"
				memoryRequest = "200Mi"
			} else {
				cpuRequest = "200m"
				memoryRequest = "800M"
			}

			obj := &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vpa-recommender",
					Namespace: namespace,
					Labels: map[string]string{
						"app":                 "vpa-recommender",
						"gardener.cloud/role": "vpa",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas:             pointer.Int32(1),
					RevisionHistoryLimit: pointer.Int32(2),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "vpa-recommender",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app":                              "vpa-recommender",
								"gardener.cloud/role":              "vpa",
								"networking.gardener.cloud/to-dns": "allowed",
								"networking.resources.gardener.cloud/to-kube-apiserver-tcp-443": "allowed",
							},
						},
						Spec: corev1.PodSpec{
							PriorityClassName: valuesRecommender.PriorityClassName,
							Containers: []corev1.Container{{
								Name:            "recommender",
								Image:           imageRecommender,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command:         []string{"./recommender"},
								Args: []string{
									"--v=3",
									"--stderrthreshold=info",
									"--pod-recommendation-min-cpu-millicores=5",
									"--pod-recommendation-min-memory-mb=10",
									fmt.Sprintf("--recommendation-margin-fraction=%s", flagRecommendationMarginFraction),
									fmt.Sprintf("--recommender-interval=%s", flagRecommenderIntervalValue),
									"--kube-api-qps=100",
									"--kube-api-burst=120",
									"--memory-saver=true",
								},
								LivenessProbe: livenessProbeVpa,
								Ports: []corev1.ContainerPort{
									{
										Name:          "server",
										ContainerPort: 8080,
									},
									{
										Name:          "metrics",
										ContainerPort: 8942,
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse(cpuRequest),
										corev1.ResourceMemory: resource.MustParse(memoryRequest),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("4Gi"),
									},
								},
							}},
						},
					},
				},
			}

			if withServiceAccount {
				obj.Spec.Template.Spec.ServiceAccountName = serviceAccountRecommender.Name
			} else {
				obj.Labels["gardener.cloud/role"] = "controlplane"
				obj.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)
				obj.Spec.Template.Spec.Containers[0].Command = append(obj.Spec.Template.Spec.Containers[0].Command, "--kubeconfig="+pathGenericKubeconfig)

				Expect(gardenerutils.InjectGenericKubeconfig(obj, genericTokenKubeconfigSecretName, shootAccessSecretRecommender.Name)).To(Succeed())
			}

			return obj
		}
		vpaRecommender = &vpaautoscalingv1.VerticalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling.k8s.io/v1",
				Kind:       "VerticalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-recommender",
				Namespace: namespace,
			},
			Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscalingv1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "vpa-recommender",
				},
				UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{UpdateMode: &vpaUpdateModeAuto},
				ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
					ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
						{
							ContainerName:    "*",
							ControlledValues: &vpaControlledValues,
							MinAllowed: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("40Mi"),
							},
						},
					},
				},
			},
		}

		serviceAccountAdmissionController = &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-admission-controller",
				Namespace: namespace,
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		clusterRoleAdmissionController = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:admission-controller",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "configmaps", "nodes", "limitranges"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"mutatingwebhookconfigurations"},
					Verbs:     []string{"create", "delete", "get", "list"},
				},
				{
					APIGroups: []string{"poc.autoscaling.k8s.io"},
					Resources: []string{"verticalpodautoscalers"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"autoscaling.k8s.io"},
					Resources: []string{"verticalpodautoscalers"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"create", "update", "get", "list", "watch"},
				},
			},
		}
		clusterRoleBindingAdmissionController = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:admission-controller",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
				Annotations: map[string]string{
					"resources.gardener.cloud/delete-on-invalid-update": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:vpa:target:admission-controller",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "vpa-admission-controller",
				Namespace: namespace,
			}},
		}
		shootAccessSecretAdmissionController = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-access-vpa-admission-controller",
				Namespace: namespace,
				Labels: map[string]string{
					"resources.gardener.cloud/purpose": "token-requestor",
				},
				Annotations: map[string]string{
					"serviceaccount.resources.gardener.cloud/name":      "vpa-admission-controller",
					"serviceaccount.resources.gardener.cloud/namespace": "kube-system",
				},
			},
			Type: corev1.SecretTypeOpaque,
		}
		serviceAdmissionControllerFor = func(clusterType component.ClusterType, topologyAwareRoutingEnabled bool) *corev1.Service {
			obj := &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vpa-webhook",
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{"app": "vpa-admission-controller"},
					Ports: []corev1.ServicePort{{
						Port:       443,
						TargetPort: intstr.FromInt(10250),
					}},
				},
			}

			if clusterType == "seed" {
				metav1.SetMetaDataAnnotation(&obj.ObjectMeta, "networking.resources.gardener.cloud/from-world-to-ports", `[{"protocol":"TCP","port":10250}]`)
			}
			if clusterType == "shoot" {
				metav1.SetMetaDataAnnotation(&obj.ObjectMeta, "networking.resources.gardener.cloud/from-all-webhook-targets-allowed-ports", `[{"protocol":"TCP","port":10250}]`)
			}

			if topologyAwareRoutingEnabled {
				metav1.SetMetaDataAnnotation(&obj.ObjectMeta, "service.kubernetes.io/topology-aware-hints", "auto")
				metav1.SetMetaDataLabel(&obj.ObjectMeta, "endpoint-slice-hints.resources.gardener.cloud/consider", "true")
			}

			return obj
		}
		deploymentAdmissionControllerFor = func(withServiceAccount bool, clusterType component.ClusterType) *appsv1.Deployment {
			obj := &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vpa-admission-controller",
					Namespace: namespace,
					Labels: map[string]string{
						"app":                 "vpa-admission-controller",
						"gardener.cloud/role": "vpa",
						"high-availability-config.resources.gardener.cloud/type": "server",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas:             pointer.Int32(1),
					RevisionHistoryLimit: pointer.Int32(2),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "vpa-admission-controller",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app":                              "vpa-admission-controller",
								"gardener.cloud/role":              "vpa",
								"networking.gardener.cloud/to-dns": "allowed",
								"networking.resources.gardener.cloud/to-kube-apiserver-tcp-443": "allowed",
							},
						},
						Spec: corev1.PodSpec{
							PriorityClassName: valuesAdmissionController.PriorityClassName,
							Containers: []corev1.Container{{
								Name:            "admission-controller",
								Image:           imageAdmissionController,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command:         []string{"./admission-controller"},
								Args: []string{
									"--v=2",
									"--stderrthreshold=info",
									"--client-ca-file=/etc/tls-certs/bundle.crt",
									"--tls-cert-file=/etc/tls-certs/tls.crt",
									"--tls-private-key=/etc/tls-certs/tls.key",
									"--address=:8944",
									"--port=10250",
									"--register-webhook=false",
								},
								LivenessProbe: livenessProbeVpa,
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("200Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("3Gi"),
									},
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "metrics",
										ContainerPort: 8944,
									},
									{
										ContainerPort: 10250,
									},
								},
								VolumeMounts: []corev1.VolumeMount{{
									Name:      "vpa-tls-certs",
									MountPath: "/etc/tls-certs",
									ReadOnly:  true,
								}},
							}},
							Volumes: []corev1.Volume{{
								Name: "vpa-tls-certs",
								VolumeSource: corev1.VolumeSource{
									Projected: &corev1.ProjectedVolumeSource{
										DefaultMode: pointer.Int32(420),
										Sources: []corev1.VolumeProjection{
											{
												Secret: &corev1.SecretProjection{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "ca",
													},
													Items: []corev1.KeyToPath{{
														Key:  "bundle.crt",
														Path: "bundle.crt",
													}},
												},
											},
											{
												Secret: &corev1.SecretProjection{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "vpa-admission-controller-server",
													},
													Items: []corev1.KeyToPath{
														{
															Key:  "tls.crt",
															Path: "tls.crt",
														},
														{
															Key:  "tls.key",
															Path: "tls.key",
														},
													},
												},
											},
										},
									},
								},
							}},
						},
					},
				},
			}

			if withServiceAccount {
				obj.Spec.Template.Spec.ServiceAccountName = serviceAccountAdmissionController.Name
				obj.Spec.Template.Spec.Containers[0].Env = append(obj.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
					Name: "NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				})
			} else {
				obj.Labels["gardener.cloud/role"] = "controlplane"
				obj.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)
				obj.Spec.Template.Spec.Containers[0].Env = append(obj.Spec.Template.Spec.Containers[0].Env,
					corev1.EnvVar{
						Name:  "KUBERNETES_SERVICE_HOST",
						Value: "kube-apiserver",
					},
					corev1.EnvVar{
						Name:  "KUBERNETES_SERVICE_PORT",
						Value: strconv.Itoa(443),
					},
				)
				obj.Spec.Template.Spec.Volumes = append(obj.Spec.Template.Spec.Volumes, corev1.Volume{
					Name: "shoot-access",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							DefaultMode: pointer.Int32(420),
							Sources: []corev1.VolumeProjection{
								{
									Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "ca",
										},
										Items: []corev1.KeyToPath{{
											Key:  "bundle.crt",
											Path: "ca.crt",
										}},
									},
								},
								{
									Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "shoot-access-vpa-admission-controller",
										},
										Items: []corev1.KeyToPath{{
											Key:  "token",
											Path: "token",
										}},
										Optional: pointer.Bool(false),
									},
								},
							},
						},
					},
				})
				obj.Spec.Template.Spec.Containers[0].VolumeMounts = append(obj.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      "shoot-access",
					MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					ReadOnly:  true,
				})
			}

			return obj
		}
		podDisruptionBudgetAdmissionController = &policyv1.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy/v1",
				Kind:       "PodDisruptionBudget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-admission-controller",
				Namespace: namespace,
				Labels:    map[string]string{"gardener.cloud/role": "vpa"},
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "vpa-admission-controller",
					},
				},
			},
		}
		vpaAdmissionController = &vpaautoscalingv1.VerticalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling.k8s.io/v1",
				Kind:       "VerticalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vpa-admission-controller",
				Namespace: namespace,
			},
			Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscalingv1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "vpa-admission-controller",
				},
				UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{UpdateMode: &vpaUpdateModeAuto},
				ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
					ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
						{
							ContainerName:    "*",
							ControlledValues: &vpaControlledValues,
							MinAllowed: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
					},
				},
			},
		}

		clusterRoleGeneralActor = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:actor",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "nodes", "limitranges"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch", "create"},
				},
				{
					APIGroups: []string{"poc.autoscaling.k8s.io"},
					Resources: []string{"verticalpodautoscalers"},
					Verbs:     []string{"get", "list", "watch", "patch"},
				},
				{
					APIGroups: []string{"autoscaling.k8s.io"},
					Resources: []string{"verticalpodautoscalers"},
					Verbs:     []string{"get", "list", "watch", "patch"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}
		clusterRoleBindingGeneralActor = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:actor",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
				Annotations: map[string]string{
					"resources.gardener.cloud/delete-on-invalid-update": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:vpa:target:actor",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "vpa-recommender",
					Namespace: namespace,
				},
				{
					Kind:      "ServiceAccount",
					Name:      "vpa-updater",
					Namespace: namespace,
				},
			},
		}
		clusterRoleGeneralTargetReader = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:target-reader",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*/scale"},
					Verbs:     []string{"get", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"replicationcontrollers"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"batch"},
					Resources: []string{"jobs", "cronjobs"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"druid.gardener.cloud"},
					Resources: []string{"etcds", "etcds/scale"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}
		clusterRoleBindingGeneralTargetReader = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:vpa:target:target-reader",
				Labels: map[string]string{
					"gardener.cloud/role": "vpa",
				},
				Annotations: map[string]string{
					"resources.gardener.cloud/delete-on-invalid-update": "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:vpa:target:target-reader",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "vpa-admission-controller",
					Namespace: namespace,
				},
				{
					Kind:      "ServiceAccount",
					Name:      "vpa-recommender",
					Namespace: namespace,
				},
				{
					Kind:      "ServiceAccount",
					Name:      "vpa-updater",
					Namespace: namespace,
				},
			},
		}
		mutatingWebhookConfiguration = &admissionregistrationv1.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admissionregistration.k8s.io/v1",
				Kind:       "MutatingWebhookConfiguration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "vpa-webhook-config-target",
				Labels: map[string]string{"remediation.webhook.shoot.gardener.cloud/exclude": "true"},
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{{
				Name:                    "vpa.k8s.io",
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL: pointer.String(fmt.Sprintf("https://vpa-webhook.%s:443", namespace)),
				},
				FailurePolicy:      &webhookFailurePolicy,
				MatchPolicy:        &webhookMatchPolicy,
				ReinvocationPolicy: &webhookReinvocationPolicy,
				SideEffects:        &webhookSideEffects,
				TimeoutSeconds:     pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &webhookScope,
						},
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
					},
					{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"autoscaling.k8s.io"},
							APIVersions: []string{"*"},
							Resources:   []string{"verticalpodautoscalers"},
							Scope:       &webhookScope,
						},
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
					},
				},
			}},
		}
	})

	JustBeforeEach(func() {
		managedResource = &resourcesv1alpha1.ManagedResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedResourceName,
				Namespace: namespace,
			},
		}
		managedResourceSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managedresource-" + managedResource.Name,
				Namespace: namespace,
			},
		}
	})

	Describe("#Deploy", func() {
		Context("cluster type seed", func() {
			BeforeEach(func() {
				vpa = New(c, namespace, sm, Values{
					ClusterType:              component.ClusterTypeSeed,
					Enabled:                  true,
					SecretNameServerCA:       secretNameCA,
					RuntimeKubernetesVersion: runtimeKubernetesVersion,
					AdmissionController:      valuesAdmissionController,
					Recommender:              valuesRecommender,
					Updater:                  valuesUpdater,
				})
				managedResourceName = "vpa"
			})

			It("should successfully deploy all resources", func() {
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))

				Expect(vpa.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(managedResource).To(DeepEqual(&resourcesv1alpha1.ManagedResource{
					TypeMeta: metav1.TypeMeta{
						APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
						Kind:       "ManagedResource",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            managedResourceName,
						Namespace:       namespace,
						Labels:          map[string]string{"gardener.cloud/role": "seed-system-component"},
						ResourceVersion: "1",
					},
					Spec: resourcesv1alpha1.ManagedResourceSpec{
						Class: pointer.String("seed"),
						SecretRefs: []corev1.LocalObjectReference{{
							Name: managedResourceSecret.Name,
						}},
						KeepObjects: pointer.Bool(false),
					},
				}))

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
				Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
				Expect(managedResourceSecret.Data).To(HaveLen(24))

				By("Verify vpa-updater resources")
				clusterRoleUpdater.Name = replaceTargetSubstrings(clusterRoleUpdater.Name)
				clusterRoleBindingUpdater.Name = replaceTargetSubstrings(clusterRoleBindingUpdater.Name)
				clusterRoleBindingUpdater.RoleRef.Name = replaceTargetSubstrings(clusterRoleBindingUpdater.RoleRef.Name)

				deploymentUpdater := deploymentUpdaterFor(true, nil, nil, nil, nil, nil, component.ClusterTypeSeed)
				adaptNetworkPolicyLabelsForClusterTypeSeed(deploymentUpdater.Spec.Template.Labels)

				Expect(string(managedResourceSecret.Data["serviceaccount__"+namespace+"__vpa-updater.yaml"])).To(Equal(componenttest.Serialize(serviceAccountUpdater)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_source_evictioner.yaml"])).To(Equal(componenttest.Serialize(clusterRoleUpdater)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_source_evictioner.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingUpdater)))
				Expect(string(managedResourceSecret.Data["deployment__"+namespace+"__vpa-updater.yaml"])).To(Equal(componenttest.Serialize(deploymentUpdater)))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__"+namespace+"__vpa-updater.yaml"])).To(Equal(componenttest.Serialize(vpaUpdater)))

				By("Verify vpa-recommender resources")
				clusterRoleRecommenderMetricsReader.Name = replaceTargetSubstrings(clusterRoleRecommenderMetricsReader.Name)
				clusterRoleBindingRecommenderMetricsReader.Name = replaceTargetSubstrings(clusterRoleBindingRecommenderMetricsReader.Name)
				clusterRoleBindingRecommenderMetricsReader.RoleRef.Name = replaceTargetSubstrings(clusterRoleBindingRecommenderMetricsReader.RoleRef.Name)
				clusterRoleRecommenderCheckpointActor.Name = replaceTargetSubstrings(clusterRoleRecommenderCheckpointActor.Name)
				clusterRoleBindingRecommenderCheckpointActor.Name = replaceTargetSubstrings(clusterRoleBindingRecommenderCheckpointActor.Name)
				clusterRoleBindingRecommenderCheckpointActor.RoleRef.Name = replaceTargetSubstrings(clusterRoleBindingRecommenderCheckpointActor.RoleRef.Name)

				deploymentRecommender := deploymentRecommenderFor(true, nil, nil, component.ClusterTypeSeed)
				adaptNetworkPolicyLabelsForClusterTypeSeed(deploymentRecommender.Spec.Template.Labels)

				Expect(string(managedResourceSecret.Data["serviceaccount__"+namespace+"__vpa-recommender.yaml"])).To(Equal(componenttest.Serialize(serviceAccountRecommender)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_source_metrics-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleRecommenderMetricsReader)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_source_metrics-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingRecommenderMetricsReader)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_source_checkpoint-actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleRecommenderCheckpointActor)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_source_checkpoint-actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingRecommenderCheckpointActor)))
				Expect(string(managedResourceSecret.Data["deployment__"+namespace+"__vpa-recommender.yaml"])).To(Equal(componenttest.Serialize(deploymentRecommender)))
				Expect(string(managedResourceSecret.Data["service__"+namespace+"__vpa-recommender.yaml"])).To(Equal(componenttest.Serialize(serviceRecommenderFor(component.ClusterTypeSeed))))
				Expect(managedResourceSecret.Data).NotTo(HaveKey("verticalpodautoscaler__" + namespace + "__vpa-recommender.yaml"))

				By("Verify vpa-admission-controller resources")
				clusterRoleAdmissionController.Name = replaceTargetSubstrings(clusterRoleAdmissionController.Name)
				clusterRoleBindingAdmissionController.Name = replaceTargetSubstrings(clusterRoleBindingAdmissionController.Name)
				clusterRoleBindingAdmissionController.RoleRef.Name = replaceTargetSubstrings(clusterRoleBindingAdmissionController.RoleRef.Name)

				deploymentAdmissionController := deploymentAdmissionControllerFor(true, component.ClusterTypeSeed)
				adaptNetworkPolicyLabelsForClusterTypeSeed(deploymentAdmissionController.Spec.Template.Labels)

				Expect(string(managedResourceSecret.Data["serviceaccount__"+namespace+"__vpa-admission-controller.yaml"])).To(Equal(componenttest.Serialize(serviceAccountAdmissionController)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_source_admission-controller.yaml"])).To(Equal(componenttest.Serialize(clusterRoleAdmissionController)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_source_admission-controller.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingAdmissionController)))
				Expect(string(managedResourceSecret.Data["service__"+namespace+"__vpa-webhook.yaml"])).To(Equal(componenttest.Serialize(serviceAdmissionControllerFor(component.ClusterTypeSeed, false))))
				Expect(string(managedResourceSecret.Data["deployment__"+namespace+"__vpa-admission-controller.yaml"])).To(Equal(componenttest.Serialize(deploymentAdmissionController)))
				Expect(string(managedResourceSecret.Data["poddisruptionbudget__"+namespace+"__vpa-admission-controller.yaml"])).To(Equal(componenttest.Serialize(podDisruptionBudgetAdmissionController)))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__"+namespace+"__vpa-admission-controller.yaml"])).To(Equal(componenttest.Serialize(vpaAdmissionController)))

				By("Verify general resources")
				clusterRoleGeneralActor.Name = replaceTargetSubstrings(clusterRoleGeneralActor.Name)
				clusterRoleGeneralTargetReader.Name = replaceTargetSubstrings(clusterRoleGeneralTargetReader.Name)
				clusterRoleBindingGeneralActor.Name = replaceTargetSubstrings(clusterRoleBindingGeneralActor.Name)
				clusterRoleBindingGeneralActor.RoleRef.Name = replaceTargetSubstrings(clusterRoleBindingGeneralActor.RoleRef.Name)
				clusterRoleBindingGeneralTargetReader.Name = replaceTargetSubstrings(clusterRoleBindingGeneralTargetReader.Name)
				clusterRoleBindingGeneralTargetReader.RoleRef.Name = replaceTargetSubstrings(clusterRoleBindingGeneralTargetReader.RoleRef.Name)
				mutatingWebhookConfiguration.Name = strings.Replace(mutatingWebhookConfiguration.Name, "-target", "-source", -1)
				mutatingWebhookConfiguration.Webhooks[0].ClientConfig = admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      "vpa-webhook",
						Namespace: namespace,
						Port:      pointer.Int32(443),
					},
				}

				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_source_actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGeneralActor)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_source_actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingGeneralActor)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_source_target-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGeneralTargetReader)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_source_target-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingGeneralTargetReader)))
				Expect(string(managedResourceSecret.Data["mutatingwebhookconfiguration____vpa-webhook-config-source.yaml"])).To(Equal(componenttest.Serialize(mutatingWebhookConfiguration)))
			})

			It("should successfully deploy with special configuration", func() {
				valuesRecommender.Interval = &metav1.Duration{Duration: 3 * time.Hour}
				valuesRecommender.RecommendationMarginFraction = pointer.Float64(8.91)

				valuesUpdater.Interval = &metav1.Duration{Duration: 4 * time.Hour}
				valuesUpdater.EvictAfterOOMThreshold = &metav1.Duration{Duration: 5 * time.Hour}
				valuesUpdater.EvictionRateBurst = pointer.Int32(1)
				valuesUpdater.EvictionRateLimit = pointer.Float64(2.34)
				valuesUpdater.EvictionTolerance = pointer.Float64(5.67)

				vpa = New(c, namespace, sm, Values{
					ClusterType:              component.ClusterTypeSeed,
					Enabled:                  true,
					SecretNameServerCA:       secretNameCA,
					RuntimeKubernetesVersion: runtimeKubernetesVersion,
					AdmissionController:      valuesAdmissionController,
					Recommender:              valuesRecommender,
					Updater:                  valuesUpdater,
				})

				Expect(vpa.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(managedResource).To(DeepEqual(&resourcesv1alpha1.ManagedResource{
					TypeMeta: metav1.TypeMeta{
						APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
						Kind:       "ManagedResource",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            managedResourceName,
						Namespace:       namespace,
						Labels:          map[string]string{"gardener.cloud/role": "seed-system-component"},
						ResourceVersion: "1",
					},
					Spec: resourcesv1alpha1.ManagedResourceSpec{
						Class: pointer.String("seed"),
						SecretRefs: []corev1.LocalObjectReference{{
							Name: managedResourceSecret.Name,
						}},
						KeepObjects: pointer.Bool(false),
					},
				}))

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

				deploymentUpdater := deploymentUpdaterFor(
					true,
					valuesUpdater.Interval,
					valuesUpdater.EvictAfterOOMThreshold,
					valuesUpdater.EvictionRateBurst,
					valuesUpdater.EvictionRateLimit,
					valuesUpdater.EvictionTolerance,
					component.ClusterTypeSeed,
				)
				adaptNetworkPolicyLabelsForClusterTypeSeed(deploymentUpdater.Spec.Template.Labels)

				deploymentRecommender := deploymentRecommenderFor(
					true,
					valuesRecommender.Interval,
					valuesRecommender.RecommendationMarginFraction,
					component.ClusterTypeSeed,
				)
				adaptNetworkPolicyLabelsForClusterTypeSeed(deploymentRecommender.Spec.Template.Labels)

				Expect(string(managedResourceSecret.Data["deployment__"+namespace+"__vpa-updater.yaml"])).To(Equal(componenttest.Serialize(deploymentUpdater)))
				Expect(string(managedResourceSecret.Data["deployment__"+namespace+"__vpa-recommender.yaml"])).To(Equal(componenttest.Serialize(deploymentRecommender)))
			})
		})

		Context("cluster type shoot", func() {
			BeforeEach(func() {
				vpa = New(c, namespace, sm, Values{
					ClusterType:              component.ClusterTypeShoot,
					Enabled:                  true,
					SecretNameServerCA:       secretNameCA,
					RuntimeKubernetesVersion: runtimeKubernetesVersion,
					AdmissionController:      valuesAdmissionController,
					Recommender:              valuesRecommender,
					Updater:                  valuesUpdater,
				})
				managedResourceName = "shoot-core-vpa"
			})

			It("should successfully deploy all resources", func() {
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))

				Expect(vpa.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(managedResource).To(DeepEqual(&resourcesv1alpha1.ManagedResource{
					TypeMeta: metav1.TypeMeta{
						APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
						Kind:       "ManagedResource",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            managedResourceName,
						Namespace:       namespace,
						ResourceVersion: "1",
						Labels:          map[string]string{"origin": "gardener"},
					},
					Spec: resourcesv1alpha1.ManagedResourceSpec{
						InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
						SecretRefs: []corev1.LocalObjectReference{{
							Name: managedResourceSecret.Name,
						}},
						KeepObjects: pointer.Bool(false),
					},
				}))

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
				Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
				Expect(managedResourceSecret.Data).To(HaveLen(15))

				By("Verify vpa-updater application resources")
				clusterRoleBindingUpdater.Subjects[0].Namespace = "kube-system"

				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_target_evictioner.yaml"])).To(Equal(componenttest.Serialize(clusterRoleUpdater)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_target_evictioner.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingUpdater)))

				By("Verify vpa-updater runtime resources")
				secret := &corev1.Secret{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(shootAccessSecretUpdater), secret)).To(Succeed())
				shootAccessSecretUpdater.ResourceVersion = "1"
				Expect(secret).To(Equal(shootAccessSecretUpdater))

				deployment := &appsv1.Deployment{}
				Expect(c.Get(ctx, kubernetesutils.Key(namespace, "vpa-updater"), deployment)).To(Succeed())
				deploymentUpdater := deploymentUpdaterFor(false, nil, nil, nil, nil, nil, component.ClusterTypeShoot)
				deploymentUpdater.ResourceVersion = "1"
				Expect(deployment).To(Equal(deploymentUpdater))

				vpa := &vpaautoscalingv1.VerticalPodAutoscaler{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(vpaUpdater), vpa)).To(Succeed())
				vpaUpdater.ResourceVersion = "1"
				Expect(vpa).To(Equal(vpaUpdater))

				By("Verify vpa-recommender application resources")
				clusterRoleBindingRecommenderMetricsReader.Subjects[0].Namespace = "kube-system"
				clusterRoleBindingRecommenderCheckpointActor.Subjects[0].Namespace = "kube-system"

				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_target_metrics-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleRecommenderMetricsReader)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_target_metrics-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingRecommenderMetricsReader)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_target_checkpoint-actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleRecommenderCheckpointActor)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_target_checkpoint-actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingRecommenderCheckpointActor)))

				By("Verify vpa-recommender runtime resources")
				secret = &corev1.Secret{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(shootAccessSecretRecommender), secret)).To(Succeed())
				shootAccessSecretRecommender.ResourceVersion = "1"
				Expect(secret).To(Equal(shootAccessSecretRecommender))

				deployment = &appsv1.Deployment{}
				Expect(c.Get(ctx, kubernetesutils.Key(namespace, "vpa-recommender"), deployment)).To(Succeed())
				deploymentRecommender := deploymentRecommenderFor(false, nil, nil, component.ClusterTypeShoot)
				deploymentRecommender.ResourceVersion = "1"
				Expect(deployment).To(Equal(deploymentRecommender))

				service := &corev1.Service{}
				Expect(c.Get(ctx, kubernetesutils.Key(namespace, "vpa-recommender"), service)).To(Succeed())
				serviceRecommender := serviceRecommenderFor(component.ClusterTypeShoot)
				serviceRecommender.ResourceVersion = "1"
				Expect(service).To(Equal(serviceRecommender))

				vpa = &vpaautoscalingv1.VerticalPodAutoscaler{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(vpaRecommender), vpa)).To(Succeed())
				vpaRecommender.ResourceVersion = "1"
				Expect(vpa).To(Equal(vpaRecommender))

				By("Verify vpa-admission-controller application resources")
				clusterRoleBindingAdmissionController.Subjects[0].Namespace = "kube-system"

				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_target_admission-controller.yaml"])).To(Equal(componenttest.Serialize(clusterRoleAdmissionController)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_target_admission-controller.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingAdmissionController)))

				By("Verify vpa-admission-controller runtime resources")
				secret = &corev1.Secret{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(shootAccessSecretAdmissionController), secret)).To(Succeed())
				shootAccessSecretAdmissionController.ResourceVersion = "1"
				Expect(secret).To(Equal(shootAccessSecretAdmissionController))

				service = &corev1.Service{}
				Expect(c.Get(ctx, kubernetesutils.Key(namespace, "vpa-webhook"), service)).To(Succeed())
				serviceAdmissionController := serviceAdmissionControllerFor(component.ClusterTypeShoot, false)
				serviceAdmissionController.ResourceVersion = "1"
				Expect(service).To(Equal(serviceAdmissionController))

				deployment = &appsv1.Deployment{}
				Expect(c.Get(ctx, kubernetesutils.Key(namespace, "vpa-admission-controller"), deployment)).To(Succeed())
				deploymentAdmissionController := deploymentAdmissionControllerFor(false, component.ClusterTypeShoot)
				deploymentAdmissionController.ResourceVersion = "1"
				Expect(deployment).To(Equal(deploymentAdmissionController))

				podDisruptionBudget := &policyv1.PodDisruptionBudget{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudgetAdmissionController), podDisruptionBudget)).To(Succeed())
				podDisruptionBudgetAdmissionController.ResourceVersion = "1"
				Expect(podDisruptionBudget).To(Equal(podDisruptionBudgetAdmissionController))

				vpa = &vpaautoscalingv1.VerticalPodAutoscaler{}
				Expect(c.Get(ctx, client.ObjectKeyFromObject(vpaAdmissionController), vpa)).To(Succeed())
				vpaAdmissionController.ResourceVersion = "1"
				Expect(vpa).To(Equal(vpaAdmissionController))

				By("Verify general application resources")
				clusterRoleBindingGeneralActor.Subjects[0].Namespace = "kube-system"
				clusterRoleBindingGeneralActor.Subjects[1].Namespace = "kube-system"
				clusterRoleBindingGeneralTargetReader.Subjects[0].Namespace = "kube-system"
				clusterRoleBindingGeneralTargetReader.Subjects[1].Namespace = "kube-system"
				clusterRoleBindingGeneralTargetReader.Subjects[2].Namespace = "kube-system"

				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_target_actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGeneralActor)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_target_actor.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingGeneralActor)))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_vpa_target_target-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGeneralTargetReader)))
				Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_vpa_target_target-reader.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingGeneralTargetReader)))
				Expect(string(managedResourceSecret.Data["mutatingwebhookconfiguration____vpa-webhook-config-target.yaml"])).To(Equal(componenttest.Serialize(mutatingWebhookConfiguration)))
				Expect(string(managedResourceSecret.Data["crd-verticalpodautoscalercheckpoints.yaml"])).To(Equal(crdVPACheckpoints))
				Expect(string(managedResourceSecret.Data["crd-verticalpodautoscalers.yaml"])).To(Equal(crdVPA))
			})

			Context("when TopologyAwareRoutingEnabled=true", func() {
				It("should successfully deploy with expected vpa-webhook service annotations and labels", func() {
					valuesAdmissionController.TopologyAwareRoutingEnabled = true
					vpa = New(c, namespace, sm, Values{
						ClusterType:              component.ClusterTypeShoot,
						Enabled:                  true,
						SecretNameServerCA:       secretNameCA,
						RuntimeKubernetesVersion: runtimeKubernetesVersion,
						AdmissionController:      valuesAdmissionController,
						Recommender:              valuesRecommender,
						Updater:                  valuesUpdater,
					})

					Expect(vpa.Deploy(ctx)).To(Succeed())

					service := &corev1.Service{}
					Expect(c.Get(ctx, kubernetesutils.Key(namespace, "vpa-webhook"), service)).To(Succeed())
					serviceAdmissionController := serviceAdmissionControllerFor(component.ClusterTypeShoot, true)
					serviceAdmissionController.ResourceVersion = "1"
					Expect(service).To(Equal(serviceAdmissionController))
				})
			})
		})
	})

	Describe("#Destroy", func() {
		Context("cluster type seed", func() {
			BeforeEach(func() {
				vpa = New(c, namespace, nil, Values{ClusterType: component.ClusterTypeSeed, RuntimeKubernetesVersion: runtimeKubernetesVersion})
				managedResourceName = "vpa"
			})

			It("should successfully destroy all resources", func() {
				Expect(c.Create(ctx, managedResource)).To(Succeed())
				Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

				Expect(vpa.Destroy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))
			})
		})

		Context("cluster type shoot", func() {
			BeforeEach(func() {
				vpa = New(c, namespace, nil, Values{ClusterType: component.ClusterTypeShoot, RuntimeKubernetesVersion: runtimeKubernetesVersion})
				managedResourceName = "shoot-core-vpa"
			})

			It("should successfully destroy all resources", func() {
				Expect(c.Create(ctx, managedResource)).To(Succeed())
				Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

				By("Create vpa-updater runtime resources")
				Expect(c.Create(ctx, deploymentUpdaterFor(true, nil, nil, nil, nil, nil, component.ClusterTypeShoot))).To(Succeed())
				Expect(c.Create(ctx, vpaUpdater)).To(Succeed())

				By("Create vpa-recommender runtime resources")
				Expect(c.Create(ctx, deploymentRecommenderFor(true, nil, nil, component.ClusterTypeShoot))).To(Succeed())
				Expect(c.Create(ctx, serviceRecommenderFor(component.ClusterTypeShoot))).To(Succeed())
				Expect(c.Create(ctx, vpaRecommender)).To(Succeed())

				By("Create vpa-admission-controller runtime resources")
				Expect(c.Create(ctx, serviceAdmissionControllerFor(component.ClusterTypeSeed, false))).To(Succeed())
				Expect(c.Create(ctx, deploymentAdmissionControllerFor(true, component.ClusterTypeShoot))).To(Succeed())
				Expect(c.Create(ctx, podDisruptionBudgetAdmissionController)).To(Succeed())
				Expect(c.Create(ctx, vpaAdmissionController)).To(Succeed())

				Expect(vpa.Destroy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))

				By("Verify vpa-updater runtime resources")
				Expect(c.Get(ctx, client.ObjectKeyFromObject(deploymentUpdaterFor(true, nil, nil, nil, nil, nil, component.ClusterTypeShoot)), &appsv1.Deployment{})).To(BeNotFoundError())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(vpaUpdater), &vpaautoscalingv1.VerticalPodAutoscaler{})).To(BeNotFoundError())

				By("Verify vpa-recommender runtime resources")
				Expect(c.Get(ctx, client.ObjectKeyFromObject(deploymentRecommenderFor(true, nil, nil, component.ClusterTypeShoot)), &appsv1.Deployment{})).To(BeNotFoundError())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(serviceRecommenderFor(component.ClusterTypeShoot)), &appsv1.Deployment{})).To(BeNotFoundError())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(vpaRecommender), &vpaautoscalingv1.VerticalPodAutoscaler{})).To(BeNotFoundError())

				By("Verify vpa-admission-controller runtime resources")
				Expect(c.Get(ctx, client.ObjectKeyFromObject(serviceAdmissionControllerFor(component.ClusterTypeSeed, false)), &corev1.Service{})).To(BeNotFoundError())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(deploymentAdmissionControllerFor(true, component.ClusterTypeShoot)), &appsv1.Deployment{})).To(BeNotFoundError())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudgetAdmissionController), &policyv1.PodDisruptionBudget{})).To(BeNotFoundError())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(vpaAdmissionController), &vpaautoscalingv1.VerticalPodAutoscaler{})).To(BeNotFoundError())
			})
		})
	})

	Context("waiting functions", func() {
		var (
			fakeOps   *retryfake.Ops
			resetVars func()
		)

		BeforeEach(func() {
			fakeOps = &retryfake.Ops{MaxAttempts: 1}
			resetVars = test.WithVars(
				&retry.Until, fakeOps.Until,
				&retry.UntilTimeout, fakeOps.UntilTimeout,
			)
		})

		AfterEach(func() {
			resetVars()
		})

		Describe("#Wait", func() {
			tests := func(managedResourceName string) {
				It("should fail because reading the ManagedResource fails", func() {
					Expect(vpa.Wait(ctx)).To(MatchError(ContainSubstring("not found")))
				})

				It("should fail because the ManagedResource doesn't become healthy", func() {
					fakeOps.MaxAttempts = 2

					Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:       managedResourceName,
							Namespace:  namespace,
							Generation: 1,
						},
						Status: resourcesv1alpha1.ManagedResourceStatus{
							ObservedGeneration: 1,
							Conditions: []gardencorev1beta1.Condition{
								{
									Type:   resourcesv1alpha1.ResourcesApplied,
									Status: gardencorev1beta1.ConditionFalse,
								},
								{
									Type:   resourcesv1alpha1.ResourcesHealthy,
									Status: gardencorev1beta1.ConditionFalse,
								},
							},
						},
					})).To(Succeed())

					Expect(vpa.Wait(ctx)).To(MatchError(ContainSubstring("is not healthy")))
				})

				It("should successfully wait for the managed resource to become healthy", func() {
					fakeOps.MaxAttempts = 2

					Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:       managedResourceName,
							Namespace:  namespace,
							Generation: 1,
						},
						Status: resourcesv1alpha1.ManagedResourceStatus{
							ObservedGeneration: 1,
							Conditions: []gardencorev1beta1.Condition{
								{
									Type:   resourcesv1alpha1.ResourcesApplied,
									Status: gardencorev1beta1.ConditionTrue,
								},
								{
									Type:   resourcesv1alpha1.ResourcesHealthy,
									Status: gardencorev1beta1.ConditionTrue,
								},
							},
						},
					})).To(Succeed())

					Expect(vpa.Wait(ctx)).To(Succeed())
				})
			}

			Context("cluster type seed", func() {
				BeforeEach(func() {
					vpa = New(c, namespace, nil, Values{ClusterType: component.ClusterTypeSeed})
				})

				tests("vpa")
			})

			Context("cluster type shoot", func() {
				BeforeEach(func() {
					vpa = New(c, namespace, nil, Values{ClusterType: component.ClusterTypeShoot})
				})

				tests("shoot-core-vpa")
			})
		})

		Describe("#WaitCleanup", func() {
			tests := func(managedResourceName string) {
				It("should fail when the wait for the managed resource deletion times out", func() {
					fakeOps.MaxAttempts = 2

					managedResource := &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      managedResourceName,
							Namespace: namespace,
						},
					}

					Expect(c.Create(ctx, managedResource)).To(Succeed())

					Expect(vpa.WaitCleanup(ctx)).To(MatchError(ContainSubstring("still exists")))
				})

				It("should not return an error when it's already removed", func() {
					Expect(vpa.WaitCleanup(ctx)).To(Succeed())
				})
			}

			Context("cluster type seed", func() {
				BeforeEach(func() {
					vpa = New(c, namespace, nil, Values{ClusterType: component.ClusterTypeSeed})
				})

				tests("vpa")
			})

			Context("cluster type shoot", func() {
				BeforeEach(func() {
					vpa = New(c, namespace, nil, Values{ClusterType: component.ClusterTypeShoot})
				})

				tests("shoot-core-vpa")
			})
		})
	})
})

func replaceTargetSubstrings(in string) string {
	return strings.Replace(in, ":target:", ":source:", -1)
}

func adaptNetworkPolicyLabelsForClusterTypeSeed(labels map[string]string) {
	delete(labels, "networking.resources.gardener.cloud/to-kube-apiserver-tcp-443")
	labels["networking.gardener.cloud/to-runtime-apiserver"] = "allowed"
}

const (
	crdVPACheckpoints = `---
# Source: https://github.com/kubernetes/autoscaler/blob/vertical-pod-autoscaler-0.13.0/vertical-pod-autoscaler/deploy/vpa-v1-crd-gen.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    api-approved.kubernetes.io: https://github.com/kubernetes/kubernetes/pull/63797
    controller-gen.kubebuilder.io/version: v0.9.2
    resources.gardener.cloud/keep-object: "true"
  creationTimestamp: null
  labels:
    gardener.cloud/role: vpa
  name: verticalpodautoscalercheckpoints.autoscaling.k8s.io
spec:
  group: autoscaling.k8s.io
  names:
    kind: VerticalPodAutoscalerCheckpoint
    listKind: VerticalPodAutoscalerCheckpointList
    plural: verticalpodautoscalercheckpoints
    shortNames:
    - vpacheckpoint
    singular: verticalpodautoscalercheckpoint
  scope: Namespaced
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        description: VerticalPodAutoscalerCheckpoint is the checkpoint of the internal
          state of VPA that is used for recovery after recommender's restart.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: 'Specification of the checkpoint. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status.'
            properties:
              containerName:
                description: Name of the checkpointed container.
                type: string
              vpaObjectName:
                description: Name of the VPA object that stored VerticalPodAutoscalerCheckpoint
                  object.
                type: string
            type: object
          status:
            description: Data of the checkpoint.
            properties:
              cpuHistogram:
                description: Checkpoint of histogram for consumption of CPU.
                properties:
                  bucketWeights:
                    description: Map from bucket index to bucket weight.
                    type: object
                    x-kubernetes-preserve-unknown-fields: true
                  referenceTimestamp:
                    description: Reference timestamp for samples collected within
                      this histogram.
                    format: date-time
                    nullable: true
                    type: string
                  totalWeight:
                    description: Sum of samples to be used as denominator for weights
                      from BucketWeights.
                    type: number
                type: object
              firstSampleStart:
                description: Timestamp of the fist sample from the histograms.
                format: date-time
                nullable: true
                type: string
              lastSampleStart:
                description: Timestamp of the last sample from the histograms.
                format: date-time
                nullable: true
                type: string
              lastUpdateTime:
                description: The time when the status was last refreshed.
                format: date-time
                nullable: true
                type: string
              memoryHistogram:
                description: Checkpoint of histogram for consumption of memory.
                properties:
                  bucketWeights:
                    description: Map from bucket index to bucket weight.
                    type: object
                    x-kubernetes-preserve-unknown-fields: true
                  referenceTimestamp:
                    description: Reference timestamp for samples collected within
                      this histogram.
                    format: date-time
                    nullable: true
                    type: string
                  totalWeight:
                    description: Sum of samples to be used as denominator for weights
                      from BucketWeights.
                    type: number
                type: object
              totalSamplesCount:
                description: Total number of samples in the histograms.
                type: integer
              version:
                description: Version of the format of the stored data.
                type: string
            type: object
        type: object
    served: true
    storage: true
  - name: v1beta2
    schema:
      openAPIV3Schema:
        description: VerticalPodAutoscalerCheckpoint is the checkpoint of the internal
          state of VPA that is used for recovery after recommender's restart.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: 'Specification of the checkpoint. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status.'
            properties:
              containerName:
                description: Name of the checkpointed container.
                type: string
              vpaObjectName:
                description: Name of the VPA object that stored VerticalPodAutoscalerCheckpoint
                  object.
                type: string
            type: object
          status:
            description: Data of the checkpoint.
            properties:
              cpuHistogram:
                description: Checkpoint of histogram for consumption of CPU.
                properties:
                  bucketWeights:
                    description: Map from bucket index to bucket weight.
                    type: object
                    x-kubernetes-preserve-unknown-fields: true
                  referenceTimestamp:
                    description: Reference timestamp for samples collected within
                      this histogram.
                    format: date-time
                    nullable: true
                    type: string
                  totalWeight:
                    description: Sum of samples to be used as denominator for weights
                      from BucketWeights.
                    type: number
                type: object
              firstSampleStart:
                description: Timestamp of the fist sample from the histograms.
                format: date-time
                nullable: true
                type: string
              lastSampleStart:
                description: Timestamp of the last sample from the histograms.
                format: date-time
                nullable: true
                type: string
              lastUpdateTime:
                description: The time when the status was last refreshed.
                format: date-time
                nullable: true
                type: string
              memoryHistogram:
                description: Checkpoint of histogram for consumption of memory.
                properties:
                  bucketWeights:
                    description: Map from bucket index to bucket weight.
                    type: object
                    x-kubernetes-preserve-unknown-fields: true
                  referenceTimestamp:
                    description: Reference timestamp for samples collected within
                      this histogram.
                    format: date-time
                    nullable: true
                    type: string
                  totalWeight:
                    description: Sum of samples to be used as denominator for weights
                      from BucketWeights.
                    type: number
                type: object
              totalSamplesCount:
                description: Total number of samples in the histograms.
                type: integer
              version:
                description: Version of the format of the stored data.
                type: string
            type: object
        type: object
    served: true
    storage: false
`

	crdVPA = `---
# Source: https://github.com/kubernetes/autoscaler/blob/vertical-pod-autoscaler-0.13.0/vertical-pod-autoscaler/deploy/vpa-v1-crd-gen.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    api-approved.kubernetes.io: https://github.com/kubernetes/kubernetes/pull/63797
    controller-gen.kubebuilder.io/version: v0.9.2
    resources.gardener.cloud/keep-object: "true"
  creationTimestamp: null
  labels:
    gardener.cloud/role: vpa
  name: verticalpodautoscalers.autoscaling.k8s.io
spec:
  group: autoscaling.k8s.io
  names:
    kind: VerticalPodAutoscaler
    listKind: VerticalPodAutoscalerList
    plural: verticalpodautoscalers
    shortNames:
    - vpa
    singular: verticalpodautoscaler
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - jsonPath: .spec.updatePolicy.updateMode
      name: Mode
      type: string
    - jsonPath: .status.recommendation.containerRecommendations[0].target.cpu
      name: CPU
      type: string
    - jsonPath: .status.recommendation.containerRecommendations[0].target.memory
      name: Mem
      type: string
    - jsonPath: .status.conditions[?(@.type=='RecommendationProvided')].status
      name: Provided
      type: string
    - jsonPath: .metadata.creationTimestamp
      name: Age
      type: date
    name: v1
    schema:
      openAPIV3Schema:
        description: VerticalPodAutoscaler is the configuration for a vertical pod
          autoscaler, which automatically manages pod resources based on historical
          and real time resource utilization.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: 'Specification of the behavior of the autoscaler. More info:
              https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status.'
            properties:
              recommenders:
                description: Recommender responsible for generating recommendation
                  for this object. List should be empty (then the default recommender
                  will generate the recommendation) or contain exactly one recommender.
                items:
                  description: VerticalPodAutoscalerRecommenderSelector points to
                    a specific Vertical Pod Autoscaler recommender. In the future
                    it might pass parameters to the recommender.
                  properties:
                    name:
                      description: Name of the recommender responsible for generating
                        recommendation for this object.
                      type: string
                  required:
                  - name
                  type: object
                type: array
              resourcePolicy:
                description: Controls how the autoscaler computes recommended resources.
                  The resource policy may be used to set constraints on the recommendations
                  for individual containers. If not specified, the autoscaler computes
                  recommended resources for all containers in the pod, without additional
                  constraints.
                properties:
                  containerPolicies:
                    description: Per-container resource policies.
                    items:
                      description: ContainerResourcePolicy controls how autoscaler
                        computes the recommended resources for a specific container.
                      properties:
                        containerName:
                          description: Name of the container or DefaultContainerResourcePolicy,
                            in which case the policy is used by the containers that
                            don't have their own policy specified.
                          type: string
                        controlledResources:
                          description: Specifies the type of recommendations that
                            will be computed (and possibly applied) by VPA. If not
                            specified, the default of [ResourceCPU, ResourceMemory]
                            will be used.
                          items:
                            description: ResourceName is the name identifying various
                              resources in a ResourceList.
                            type: string
                          type: array
                        controlledValues:
                          description: Specifies which resource values should be controlled.
                            The default is "RequestsAndLimits".
                          enum:
                          - RequestsAndLimits
                          - RequestsOnly
                          type: string
                        maxAllowed:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Specifies the maximum amount of resources that
                            will be recommended for the container. The default is
                            no maximum.
                          type: object
                        minAllowed:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Specifies the minimal amount of resources that
                            will be recommended for the container. The default is
                            no minimum.
                          type: object
                        mode:
                          description: Whether autoscaler is enabled for the container.
                            The default is "Auto".
                          enum:
                          - Auto
                          - "Off"
                          type: string
                      type: object
                    type: array
                type: object
              targetRef:
                description: TargetRef points to the controller managing the set of
                  pods for the autoscaler to control - e.g. Deployment, StatefulSet.
                  VerticalPodAutoscaler can be targeted at controller implementing
                  scale subresource (the pod set is retrieved from the controller's
                  ScaleStatus) or some well known controllers (e.g. for DaemonSet
                  the pod set is read from the controller's spec). If VerticalPodAutoscaler
                  cannot use specified target it will report ConfigUnsupported condition.
                  Note that VerticalPodAutoscaler does not require full implementation
                  of scale subresource - it will not use it to modify the replica
                  count. The only thing retrieved is a label selector matching pods
                  grouped by the target resource.
                properties:
                  apiVersion:
                    description: API version of the referent
                    type: string
                  kind:
                    description: 'Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"'
                    type: string
                  name:
                    description: 'Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names'
                    type: string
                required:
                - kind
                - name
                type: object
                x-kubernetes-map-type: atomic
              updatePolicy:
                description: Describes the rules on how changes are applied to the
                  pods. If not specified, all fields in the ` + "`" + `PodUpdatePolicy` + "`" + ` are
                  set to their default values.
                properties:
                  minReplicas:
                    description: Minimal number of replicas which need to be alive
                      for Updater to attempt pod eviction (pending other checks like
                      PDB). Only positive values are allowed. Overrides global '--min-replicas'
                      flag.
                    format: int32
                    type: integer
                  updateMode:
                    description: Controls when autoscaler applies changes to the pod
                      resources. The default is 'Auto'.
                    enum:
                    - "Off"
                    - Initial
                    - Recreate
                    - Auto
                    type: string
                type: object
            required:
            - targetRef
            type: object
          status:
            description: Current information about the autoscaler.
            properties:
              conditions:
                description: Conditions is the set of conditions required for this
                  autoscaler to scale its target, and indicates whether or not those
                  conditions are met.
                items:
                  description: VerticalPodAutoscalerCondition describes the state
                    of a VerticalPodAutoscaler at a certain point.
                  properties:
                    lastTransitionTime:
                      description: lastTransitionTime is the last time the condition
                        transitioned from one status to another
                      format: date-time
                      type: string
                    message:
                      description: message is a human-readable explanation containing
                        details about the transition
                      type: string
                    reason:
                      description: reason is the reason for the condition's last transition.
                      type: string
                    status:
                      description: status is the status of the condition (True, False,
                        Unknown)
                      type: string
                    type:
                      description: type describes the current condition
                      type: string
                  required:
                  - status
                  - type
                  type: object
                type: array
              recommendation:
                description: The most recently computed amount of resources recommended
                  by the autoscaler for the controlled pods.
                properties:
                  containerRecommendations:
                    description: Resources recommended by the autoscaler for each
                      container.
                    items:
                      description: RecommendedContainerResources is the recommendation
                        of resources computed by autoscaler for a specific container.
                        Respects the container resource policy if present in the spec.
                        In particular the recommendation is not produced for containers
                        with ` + "`" + `ContainerScalingMode` + "`" + ` set to 'Off'.
                      properties:
                        containerName:
                          description: Name of the container.
                          type: string
                        lowerBound:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Minimum recommended amount of resources. Observes
                            ContainerResourcePolicy. This amount is not guaranteed
                            to be sufficient for the application to operate in a stable
                            way, however running with less resources is likely to
                            have significant impact on performance/availability.
                          type: object
                        target:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Recommended amount of resources. Observes ContainerResourcePolicy.
                          type: object
                        uncappedTarget:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: The most recent recommended resources target
                            computed by the autoscaler for the controlled pods, based
                            only on actual resource usage, not taking into account
                            the ContainerResourcePolicy. May differ from the Recommendation
                            if the actual resource usage causes the target to violate
                            the ContainerResourcePolicy (lower than MinAllowed or
                            higher that MaxAllowed). Used only as status indication,
                            will not affect actual resource assignment.
                          type: object
                        upperBound:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Maximum recommended amount of resources. Observes
                            ContainerResourcePolicy. Any resources allocated beyond
                            this value are likely wasted. This value may be larger
                            than the maximum amount of application is actually capable
                            of consuming.
                          type: object
                      required:
                      - target
                      type: object
                    type: array
                type: object
            type: object
        required:
        - spec
        type: object
    served: true
    storage: true
    subresources: {}
  - deprecated: true
    deprecationWarning: autoscaling.k8s.io/v1beta2 API is deprecated
    name: v1beta2
    schema:
      openAPIV3Schema:
        description: VerticalPodAutoscaler is the configuration for a vertical pod
          autoscaler, which automatically manages pod resources based on historical
          and real time resource utilization.
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: 'Specification of the behavior of the autoscaler. More info:
              https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status.'
            properties:
              resourcePolicy:
                description: Controls how the autoscaler computes recommended resources.
                  The resource policy may be used to set constraints on the recommendations
                  for individual containers. If not specified, the autoscaler computes
                  recommended resources for all containers in the pod, without additional
                  constraints.
                properties:
                  containerPolicies:
                    description: Per-container resource policies.
                    items:
                      description: ContainerResourcePolicy controls how autoscaler
                        computes the recommended resources for a specific container.
                      properties:
                        containerName:
                          description: Name of the container or DefaultContainerResourcePolicy,
                            in which case the policy is used by the containers that
                            don't have their own policy specified.
                          type: string
                        maxAllowed:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Specifies the maximum amount of resources that
                            will be recommended for the container. The default is
                            no maximum.
                          type: object
                        minAllowed:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Specifies the minimal amount of resources that
                            will be recommended for the container. The default is
                            no minimum.
                          type: object
                        mode:
                          description: Whether autoscaler is enabled for the container.
                            The default is "Auto".
                          enum:
                          - Auto
                          - "Off"
                          type: string
                      type: object
                    type: array
                type: object
              targetRef:
                description: TargetRef points to the controller managing the set of
                  pods for the autoscaler to control - e.g. Deployment, StatefulSet.
                  VerticalPodAutoscaler can be targeted at controller implementing
                  scale subresource (the pod set is retrieved from the controller's
                  ScaleStatus) or some well known controllers (e.g. for DaemonSet
                  the pod set is read from the controller's spec). If VerticalPodAutoscaler
                  cannot use specified target it will report ConfigUnsupported condition.
                  Note that VerticalPodAutoscaler does not require full implementation
                  of scale subresource - it will not use it to modify the replica
                  count. The only thing retrieved is a label selector matching pods
                  grouped by the target resource.
                properties:
                  apiVersion:
                    description: API version of the referent
                    type: string
                  kind:
                    description: 'Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"'
                    type: string
                  name:
                    description: 'Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names'
                    type: string
                required:
                - kind
                - name
                type: object
                x-kubernetes-map-type: atomic
              updatePolicy:
                description: Describes the rules on how changes are applied to the
                  pods. If not specified, all fields in the ` + "`" + `PodUpdatePolicy` + "`" + ` are
                  set to their default values.
                properties:
                  updateMode:
                    description: Controls when autoscaler applies changes to the pod
                      resources. The default is 'Auto'.
                    enum:
                    - "Off"
                    - Initial
                    - Recreate
                    - Auto
                    type: string
                type: object
            required:
            - targetRef
            type: object
          status:
            description: Current information about the autoscaler.
            properties:
              conditions:
                description: Conditions is the set of conditions required for this
                  autoscaler to scale its target, and indicates whether or not those
                  conditions are met.
                items:
                  description: VerticalPodAutoscalerCondition describes the state
                    of a VerticalPodAutoscaler at a certain point.
                  properties:
                    lastTransitionTime:
                      description: lastTransitionTime is the last time the condition
                        transitioned from one status to another
                      format: date-time
                      type: string
                    message:
                      description: message is a human-readable explanation containing
                        details about the transition
                      type: string
                    reason:
                      description: reason is the reason for the condition's last transition.
                      type: string
                    status:
                      description: status is the status of the condition (True, False,
                        Unknown)
                      type: string
                    type:
                      description: type describes the current condition
                      type: string
                  required:
                  - status
                  - type
                  type: object
                type: array
              recommendation:
                description: The most recently computed amount of resources recommended
                  by the autoscaler for the controlled pods.
                properties:
                  containerRecommendations:
                    description: Resources recommended by the autoscaler for each
                      container.
                    items:
                      description: RecommendedContainerResources is the recommendation
                        of resources computed by autoscaler for a specific container.
                        Respects the container resource policy if present in the spec.
                        In particular the recommendation is not produced for containers
                        with ` + "`" + `ContainerScalingMode` + "`" + ` set to 'Off'.
                      properties:
                        containerName:
                          description: Name of the container.
                          type: string
                        lowerBound:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Minimum recommended amount of resources. Observes
                            ContainerResourcePolicy. This amount is not guaranteed
                            to be sufficient for the application to operate in a stable
                            way, however running with less resources is likely to
                            have significant impact on performance/availability.
                          type: object
                        target:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Recommended amount of resources. Observes ContainerResourcePolicy.
                          type: object
                        uncappedTarget:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: The most recent recommended resources target
                            computed by the autoscaler for the controlled pods, based
                            only on actual resource usage, not taking into account
                            the ContainerResourcePolicy. May differ from the Recommendation
                            if the actual resource usage causes the target to violate
                            the ContainerResourcePolicy (lower than MinAllowed or
                            higher that MaxAllowed). Used only as status indication,
                            will not affect actual resource assignment.
                          type: object
                        upperBound:
                          additionalProperties:
                            anyOf:
                            - type: integer
                            - type: string
                            pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                            x-kubernetes-int-or-string: true
                          description: Maximum recommended amount of resources. Observes
                            ContainerResourcePolicy. Any resources allocated beyond
                            this value are likely wasted. This value may be larger
                            than the maximum amount of application is actually capable
                            of consuming.
                          type: object
                      required:
                      - target
                      type: object
                    type: array
                type: object
            type: object
        required:
        - spec
        type: object
    served: true
    storage: false
`
)