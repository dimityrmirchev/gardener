// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/rest"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/extensions/pkg/webhook"
	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	securityv1alpha1constants "github.com/gardener/gardener/pkg/apis/security/v1alpha1/constants"
	"github.com/gardener/gardener/pkg/component/nodemanagement/machinecontrollermanager"
	"github.com/gardener/gardener/pkg/provider-local/imagevector"
	"github.com/gardener/gardener/pkg/provider-local/local"
	"github.com/gardener/gardener/pkg/utils/version"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(client client.Client, restConfig *rest.Config, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		client:     client,
		restConfig: restConfig,
		logger:     logger.WithName("local-controlplane-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer

	client     client.Client
	restConfig *rest.Config
	logger     logr.Logger
}

// EnsureMachineControllerManagerDeployment ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerDeployment(ctx context.Context, gctx extensionscontextwebhook.GardenContext, newObj, _ *appsv1.Deployment) error {
	cloudProviderSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1constants.SecretNameCloudProvider,
			Namespace: newObj.Namespace,
		},
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(cloudProviderSecret), cloudProviderSecret); err != nil {
		return fmt.Errorf("failed getting cloudprovider secret: %w", err)
	}

	image, err := imagevector.ImageVector().FindImage(imagevector.ImageNameMachineControllerManagerProviderLocal)
	if err != nil {
		return err
	}

	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed reading Cluster: %w", err)
	}

	sidecarContainer := machinecontrollermanager.ProviderSidecarContainer(cluster.Shoot, newObj.GetNamespace(), local.Name, image.String())

	if cloudProviderSecret.Labels[securityv1alpha1constants.LabelPurpose] == securityv1alpha1constants.LabelPurposeWorkloadIdentityTokenRequestor {
		const volumeName = "workload-identity"
		sidecarContainer.VolumeMounts = webhook.EnsureVolumeMountWithName(sidecarContainer.VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: "/var/run/secrets/gardener.cloud/workload-identity",
		})
		newObj.Spec.Template.Spec.Volumes = webhook.EnsureVolumeWithName(newObj.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: v1beta1constants.SecretNameCloudProvider,
								},
								Items: []corev1.KeyToPath{
									{
										Key:  securityv1alpha1constants.DataKeyToken,
										Path: "token",
									},
								},
							},
						},
					},
				},
			},
		})
	}

	newObj.Spec.Template.Spec.Containers = webhook.EnsureContainerWithName(
		newObj.Spec.Template.Spec.Containers,
		sidecarContainer,
	)

	return nil
}

// EnsureMachineControllerManagerVPA ensures that the machine-controller-manager VPA conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerVPA(_ context.Context, _ extensionscontextwebhook.GardenContext, newObj, _ *vpaautoscalingv1.VerticalPodAutoscaler) error {
	newObj.Spec.ResourcePolicy.ContainerPolicies = webhook.EnsureVPAContainerResourcePolicyWithName(
		newObj.Spec.ResourcePolicy.ContainerPolicies,
		machinecontrollermanager.ProviderSidecarVPAContainerPolicy(local.Name),
	)
	return nil
}

func (e *ensurer) EnsureKubeletConfiguration(_ context.Context, _ extensionscontextwebhook.GardenContext, kubeletVersion *semver.Version, newObj, _ *kubeletconfigv1beta1.KubeletConfiguration) error {
	newObj.FailSwapOn = ptr.To(false)

	// kubelet's cgroup driver defaults to systemd already for Shoots with K8s version 1.31+.
	// No need to set the cgroup driver explicitly for 1.31+.
	if version.ConstraintK8sLess131.Check(kubeletVersion) {
		newObj.CgroupDriver = "systemd"
	}

	return nil
}

// EnsureKubeSchedulerDeployment ensures that the kube-scheduler deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeSchedulerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, newObj, _ *appsv1.Deployment) error {
	newObj.Spec.Template.Labels["injected-by"] = "provider-local"
	return nil
}
