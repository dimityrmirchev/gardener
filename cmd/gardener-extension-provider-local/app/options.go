// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package app

import (
	extensionsauditbackendcontroller "github.com/gardener/gardener/extensions/pkg/controller/auditbackend"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	extensionscontrolplanecontroller "github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	extensionsdnsrecordcontroller "github.com/gardener/gardener/extensions/pkg/controller/dnsrecord"
	extensionshealthcheckcontroller "github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	extensionsheartbeatcontroller "github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	extensionsinfrastructurecontroller "github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	extensionsoperatingsystemconfgcontroller "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsworkercontroller "github.com/gardener/gardener/extensions/pkg/controller/worker"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	extensioncontrolplanewebhook "github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	extensionshootwebhook "github.com/gardener/gardener/extensions/pkg/webhook/shoot"
	auditbackendcontroller "github.com/gardener/gardener/pkg/provider-local/controller/auditbackend"
	backupbucketcontroller "github.com/gardener/gardener/pkg/provider-local/controller/backupbucket"
	backupentrycontroller "github.com/gardener/gardener/pkg/provider-local/controller/backupentry"
	controlplanecontroller "github.com/gardener/gardener/pkg/provider-local/controller/controlplane"
	dnsrecordcontroller "github.com/gardener/gardener/pkg/provider-local/controller/dnsrecord"
	healthcheckcontroller "github.com/gardener/gardener/pkg/provider-local/controller/healthcheck"
	infrastructurecontroller "github.com/gardener/gardener/pkg/provider-local/controller/infrastructure"
	ingresscontroller "github.com/gardener/gardener/pkg/provider-local/controller/ingress"
	operatingsystemconfigcontroller "github.com/gardener/gardener/pkg/provider-local/controller/operatingsystemconfig"
	servicecontroller "github.com/gardener/gardener/pkg/provider-local/controller/service"
	workercontroller "github.com/gardener/gardener/pkg/provider-local/controller/worker"
	controlplanewebhook "github.com/gardener/gardener/pkg/provider-local/webhook/controlplane"
	controlplaneexposurewebhook "github.com/gardener/gardener/pkg/provider-local/webhook/controlplaneexposure"
	dnsconfigwebhook "github.com/gardener/gardener/pkg/provider-local/webhook/dnsconfig"
	networkpolicywebhook "github.com/gardener/gardener/pkg/provider-local/webhook/networkpolicy"
	nodewebhook "github.com/gardener/gardener/pkg/provider-local/webhook/node"
	shootwebhook "github.com/gardener/gardener/pkg/provider-local/webhook/shoot"
)

// ControllerSwitchOptions are the controllercmd.SwitchOptions for the provider controllers.
func ControllerSwitchOptions() *controllercmd.SwitchOptions {
	return controllercmd.NewSwitchOptions(
		controllercmd.Switch(extensionsauditbackendcontroller.ControllerName, auditbackendcontroller.AddToManager),
		controllercmd.Switch(backupbucketcontroller.ControllerName, backupbucketcontroller.AddToManager),
		controllercmd.Switch(backupentrycontroller.ControllerName, backupentrycontroller.AddToManager),
		controllercmd.Switch(extensionscontrolplanecontroller.ControllerName, controlplanecontroller.AddToManager),
		controllercmd.Switch(extensionsdnsrecordcontroller.ControllerName, dnsrecordcontroller.AddToManager),
		controllercmd.Switch(extensionsinfrastructurecontroller.ControllerName, infrastructurecontroller.AddToManager),
		controllercmd.Switch(extensionsworkercontroller.ControllerName, workercontroller.AddToManager),
		controllercmd.Switch(ingresscontroller.ControllerName, ingresscontroller.AddToManager),
		controllercmd.Switch(servicecontroller.ControllerName, servicecontroller.AddToManager),
		controllercmd.Switch(extensionshealthcheckcontroller.ControllerName, healthcheckcontroller.AddToManager),
		controllercmd.Switch(extensionsoperatingsystemconfgcontroller.ControllerName, operatingsystemconfigcontroller.AddToManager),
		controllercmd.Switch(extensionsheartbeatcontroller.ControllerName, extensionsheartbeatcontroller.AddToManager),
	)
}

// WebhookSwitchOptions are the webhookcmd.SwitchOptions for the provider webhooks.
func WebhookSwitchOptions() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(extensioncontrolplanewebhook.ExposureWebhookName, controlplaneexposurewebhook.AddToManager),
		webhookcmd.Switch(extensioncontrolplanewebhook.WebhookName, controlplanewebhook.AddToManager),
		webhookcmd.Switch(extensionshootwebhook.WebhookName, shootwebhook.AddToManager),
		webhookcmd.Switch(dnsconfigwebhook.WebhookName, dnsconfigwebhook.AddToManager),
		webhookcmd.Switch(networkpolicywebhook.WebhookName, networkpolicywebhook.AddToManager),
		webhookcmd.Switch(nodewebhook.WebhookName, nodewebhook.AddToManager),
		webhookcmd.Switch(nodewebhook.WebhookNameShoot, nodewebhook.AddShootWebhookToManager),
	)
}
