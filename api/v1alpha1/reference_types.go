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

type PreRenderer struct {
	// Custom non white-spaced and non alpha-numeric open delimiter used for go templating action to pre-render. For e.g., <%. Default is {{
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=2
	// +optional
	LeftDelimiter string `json:"openDelimiter,omitempty"`

	// Custom non white-spaced and non alpha-numeric close delimiter used for go templating action to pre-render. For e.g., %>. Default is  }}
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=2
	// +optional
	RightDelimiter string `json:"closeDelimiter,omitempty"`

	// Enable to not render actions within delimiters {{ }} so that they can be rendered by Helm. Defaults to false. If both helm templating
	// and pre-rendering are desired, then enable EnableHelmTemplating and specify non default open and close delimiters at LeftDelimiter and RightDelimiter respectively
	// +optional
	EnableHelmTemplating bool `json:"enableHelmTemplating,omitempty"`
}
