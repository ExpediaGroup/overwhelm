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

package v1alpha1

// SourceReference is the reference of the source where the chart is available.
// For e.g., the HelmRepository resource specifying where the helm chart is located

type PreRenderReference struct {
	// Kind of the resource that holds the values in the data stanza that are to be rendered with, valid values are ('Secret', 'ConfigMap').
	// +kubebuilder:validation:Enum=Secret;ConfigMap
	// +required
	Kind string `json:"kind"`

	// Name of the  resource that holds the values that are to be rendered with.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +required
	Name string `json:"name"`

	// Namespace where the resource resides
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Optional
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Prefix of the value to be parsed or scanned for rendering. for e.g., to render {{ xyz.environment }},
	// the PreRenderPrefix will be "xyz".
	// +required
	PreRenderPrefix string `json:"preRenderPrefix,omitempty"`

	// Optional marks this PreRenderReference as optional. When set, a not found error
	// for any of the keys is ignored, but any error will still result in a reconciliation failure.
	// Defaults to false
	// +optional
	Optional bool `json:"optional,omitempty"`
}
