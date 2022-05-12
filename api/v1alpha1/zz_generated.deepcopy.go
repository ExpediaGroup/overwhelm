//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright 2022 Expedia Group
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/pkg/apis/kustomize"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Application) DeepCopyInto(out *Application) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Application.
func (in *Application) DeepCopy() *Application {
	if in == nil {
		return nil
	}
	out := new(Application)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Application) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationList) DeepCopyInto(out *ApplicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Application, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationList.
func (in *ApplicationList) DeepCopy() *ApplicationList {
	if in == nil {
		return nil
	}
	out := new(ApplicationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationSpec) DeepCopyInto(out *ApplicationSpec) {
	*out = *in
	out.Chart = in.Chart
	out.Interval = in.Interval
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.Install != nil {
		in, out := &in.Install, &out.Install
		*out = new(v2beta1.Install)
		(*in).DeepCopyInto(*out)
	}
	if in.Upgrade != nil {
		in, out := &in.Upgrade, &out.Upgrade
		*out = new(v2beta1.Upgrade)
		(*in).DeepCopyInto(*out)
	}
	if in.Uninstall != nil {
		in, out := &in.Uninstall, &out.Uninstall
		*out = new(v2beta1.Uninstall)
		(*in).DeepCopyInto(*out)
	}
	if in.Rollback != nil {
		in, out := &in.Rollback, &out.Rollback
		*out = new(v2beta1.Rollback)
		(*in).DeepCopyInto(*out)
	}
	if in.Test != nil {
		in, out := &in.Test, &out.Test
		*out = new(v2beta1.Test)
		(*in).DeepCopyInto(*out)
	}
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PreRenderFrom != nil {
		in, out := &in.PreRenderFrom, &out.PreRenderFrom
		*out = make([]PreRenderReference, len(*in))
		copy(*out, *in)
	}
	if in.PostRenderers != nil {
		in, out := &in.PostRenderers, &out.PostRenderers
		*out = make([]PostRenderer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationSpec.
func (in *ApplicationSpec) DeepCopy() *ApplicationSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationStatus) DeepCopyInto(out *ApplicationStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationStatus.
func (in *ApplicationStatus) DeepCopy() *ApplicationStatus {
	if in == nil {
		return nil
	}
	out := new(ApplicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChart) DeepCopyInto(out *HelmChart) {
	*out = *in
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChart.
func (in *HelmChart) DeepCopy() *HelmChart {
	if in == nil {
		return nil
	}
	out := new(HelmChart)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChartSpec) DeepCopyInto(out *HelmChartSpec) {
	*out = *in
	out.SourceRef = in.SourceRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChartSpec.
func (in *HelmChartSpec) DeepCopy() *HelmChartSpec {
	if in == nil {
		return nil
	}
	out := new(HelmChartSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kustomize) DeepCopyInto(out *Kustomize) {
	*out = *in
	if in.PatchesStrategicMerge != nil {
		in, out := &in.PatchesStrategicMerge, &out.PatchesStrategicMerge
		*out = make([]v1.JSON, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PatchesJSON6902 != nil {
		in, out := &in.PatchesJSON6902, &out.PatchesJSON6902
		*out = make([]kustomize.JSON6902Patch, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kustomize.
func (in *Kustomize) DeepCopy() *Kustomize {
	if in == nil {
		return nil
	}
	out := new(Kustomize)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostRenderer) DeepCopyInto(out *PostRenderer) {
	*out = *in
	if in.Kustomize != nil {
		in, out := &in.Kustomize, &out.Kustomize
		*out = new(Kustomize)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostRenderer.
func (in *PostRenderer) DeepCopy() *PostRenderer {
	if in == nil {
		return nil
	}
	out := new(PostRenderer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreRenderReference) DeepCopyInto(out *PreRenderReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreRenderReference.
func (in *PreRenderReference) DeepCopy() *PreRenderReference {
	if in == nil {
		return nil
	}
	out := new(PreRenderReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SourceReference) DeepCopyInto(out *SourceReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SourceReference.
func (in *SourceReference) DeepCopy() *SourceReference {
	if in == nil {
		return nil
	}
	out := new(SourceReference)
	in.DeepCopyInto(out)
	return out
}
