// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

// ManagedSeedStatusApplyConfiguration represents an declarative configuration of the ManagedSeedStatus type for use
// with apply.
type ManagedSeedStatusApplyConfiguration struct {
	Conditions         []v1beta1.Condition `json:"conditions,omitempty"`
	ObservedGeneration *int64              `json:"observedGeneration,omitempty"`
}

// ManagedSeedStatusApplyConfiguration constructs an declarative configuration of the ManagedSeedStatus type for use with
// apply.
func ManagedSeedStatus() *ManagedSeedStatusApplyConfiguration {
	return &ManagedSeedStatusApplyConfiguration{}
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *ManagedSeedStatusApplyConfiguration) WithConditions(values ...v1beta1.Condition) *ManagedSeedStatusApplyConfiguration {
	for i := range values {
		b.Conditions = append(b.Conditions, values[i])
	}
	return b
}

// WithObservedGeneration sets the ObservedGeneration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ObservedGeneration field is set to the value of the last call.
func (b *ManagedSeedStatusApplyConfiguration) WithObservedGeneration(value int64) *ManagedSeedStatusApplyConfiguration {
	b.ObservedGeneration = &value
	return b
}
