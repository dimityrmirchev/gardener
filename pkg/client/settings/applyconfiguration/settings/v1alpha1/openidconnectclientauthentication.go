// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// OpenIDConnectClientAuthenticationApplyConfiguration represents an declarative configuration of the OpenIDConnectClientAuthentication type for use
// with apply.
type OpenIDConnectClientAuthenticationApplyConfiguration struct {
	Secret      *string           `json:"secret,omitempty"`
	ExtraConfig map[string]string `json:"extraConfig,omitempty"`
}

// OpenIDConnectClientAuthenticationApplyConfiguration constructs an declarative configuration of the OpenIDConnectClientAuthentication type for use with
// apply.
func OpenIDConnectClientAuthentication() *OpenIDConnectClientAuthenticationApplyConfiguration {
	return &OpenIDConnectClientAuthenticationApplyConfiguration{}
}

// WithSecret sets the Secret field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Secret field is set to the value of the last call.
func (b *OpenIDConnectClientAuthenticationApplyConfiguration) WithSecret(value string) *OpenIDConnectClientAuthenticationApplyConfiguration {
	b.Secret = &value
	return b
}

// WithExtraConfig puts the entries into the ExtraConfig field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the ExtraConfig field,
// overwriting an existing map entries in ExtraConfig field with the same key.
func (b *OpenIDConnectClientAuthenticationApplyConfiguration) WithExtraConfig(entries map[string]string) *OpenIDConnectClientAuthenticationApplyConfiguration {
	if b.ExtraConfig == nil && len(entries) > 0 {
		b.ExtraConfig = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.ExtraConfig[k] = v
	}
	return b
}
