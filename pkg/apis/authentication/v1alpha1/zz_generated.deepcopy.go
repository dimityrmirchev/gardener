//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminKubeconfigRequest) DeepCopyInto(out *AdminKubeconfigRequest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminKubeconfigRequest.
func (in *AdminKubeconfigRequest) DeepCopy() *AdminKubeconfigRequest {
	if in == nil {
		return nil
	}
	out := new(AdminKubeconfigRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AdminKubeconfigRequest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminKubeconfigRequestSpec) DeepCopyInto(out *AdminKubeconfigRequestSpec) {
	*out = *in
	if in.ExpirationSeconds != nil {
		in, out := &in.ExpirationSeconds, &out.ExpirationSeconds
		*out = new(int64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminKubeconfigRequestSpec.
func (in *AdminKubeconfigRequestSpec) DeepCopy() *AdminKubeconfigRequestSpec {
	if in == nil {
		return nil
	}
	out := new(AdminKubeconfigRequestSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminKubeconfigRequestStatus) DeepCopyInto(out *AdminKubeconfigRequestStatus) {
	*out = *in
	if in.Kubeconfig != nil {
		in, out := &in.Kubeconfig, &out.Kubeconfig
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	in.ExpirationTimestamp.DeepCopyInto(&out.ExpirationTimestamp)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminKubeconfigRequestStatus.
func (in *AdminKubeconfigRequestStatus) DeepCopy() *AdminKubeconfigRequestStatus {
	if in == nil {
		return nil
	}
	out := new(AdminKubeconfigRequestStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenRequest) DeepCopyInto(out *TokenRequest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenRequest.
func (in *TokenRequest) DeepCopy() *TokenRequest {
	if in == nil {
		return nil
	}
	out := new(TokenRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TokenRequest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenRequestSpec) DeepCopyInto(out *TokenRequestSpec) {
	*out = *in
	if in.ExpirationSeconds != nil {
		in, out := &in.ExpirationSeconds, &out.ExpirationSeconds
		*out = new(int64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenRequestSpec.
func (in *TokenRequestSpec) DeepCopy() *TokenRequestSpec {
	if in == nil {
		return nil
	}
	out := new(TokenRequestSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenRequestStatus) DeepCopyInto(out *TokenRequestStatus) {
	*out = *in
	in.ExpirationTimestamp.DeepCopyInto(&out.ExpirationTimestamp)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenRequestStatus.
func (in *TokenRequestStatus) DeepCopy() *TokenRequestStatus {
	if in == nil {
		return nil
	}
	out := new(TokenRequestStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadIdentity) DeepCopyInto(out *WorkloadIdentity) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadIdentity.
func (in *WorkloadIdentity) DeepCopy() *WorkloadIdentity {
	if in == nil {
		return nil
	}
	out := new(WorkloadIdentity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkloadIdentity) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadIdentityList) DeepCopyInto(out *WorkloadIdentityList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WorkloadIdentity, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadIdentityList.
func (in *WorkloadIdentityList) DeepCopy() *WorkloadIdentityList {
	if in == nil {
		return nil
	}
	out := new(WorkloadIdentityList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkloadIdentityList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadIdentitySpec) DeepCopyInto(out *WorkloadIdentitySpec) {
	*out = *in
	if in.Audiences != nil {
		in, out := &in.Audiences, &out.Audiences
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadIdentitySpec.
func (in *WorkloadIdentitySpec) DeepCopy() *WorkloadIdentitySpec {
	if in == nil {
		return nil
	}
	out := new(WorkloadIdentitySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadIdentityStatus) DeepCopyInto(out *WorkloadIdentityStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadIdentityStatus.
func (in *WorkloadIdentityStatus) DeepCopy() *WorkloadIdentityStatus {
	if in == nil {
		return nil
	}
	out := new(WorkloadIdentityStatus)
	in.DeepCopyInto(out)
	return out
}
