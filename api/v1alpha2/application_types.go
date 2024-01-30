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

package v1alpha2

import (
	"fmt"
	"strings"

	"github.com/ExpediaGroup/overwhelm/analyzer"
	"github.com/fluxcd/helm-controller/api/v2beta2"
	"github.com/fluxcd/pkg/apis/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const PodReady = "PodReady"
const ContainersNotReady = "ContainersNotReady"
const PodInitializing = "PodInitializing"

type Metadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ReleaseTemplate struct {
	// Metadata to be applied to the resources created by the Application Controller
	// +optional
	Metadata `json:"metadata,omitempty"`

	// Spec to be applied to the Helm Release resource created by the Application Controller
	// +required
	Spec v2beta2.HelmReleaseSpec `json:"spec,omitempty"`
}

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Data to be consolidated for the Helm Chart's values.yaml file
	// +optional
	Data map[string]string `json:"data,omitempty"`

	// PreRenderer holds custom templating delimiters and a flag.
	// By default, standard delimiters "{{" and "}}" will be used to render values within. If specified then the custom delimiters will be used.
	// +optional
	PreRenderer PreRenderer `json:"preRenderer,omitempty"`

	// Template of Release metadata and spec needed for the resources created by the Application Controller
	// +required
	Template ReleaseTemplate `json:"template,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions holds the conditions for the Application.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// HelmReleaseGeneration is the helm release generation number
	// +optional
	HelmReleaseGeneration int64 `json:"helmReleaseGeneration,omitempty"`

	// ValuesCheckSum is the checksum of the values file associated with the helm chart
	// +optional
	ValuesCheckSum string `json:"valuesCheckSum,omitempty"`

	// Failures is the reconciliation failure count against the latest desired
	// state. It is reset after a successful reconciliation.
	// +optional
	Failures int64 `json:"failures,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=app
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec,omitempty"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

func AppInProgressStatus(application *Application) {
	application.Status.Conditions = []metav1.Condition{}
	condition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  metav1.ConditionUnknown,
		Reason:  meta.ProgressingReason,
		Message: "Reconciliation in progress",
	}
	apimeta.SetStatusCondition(&application.Status.Conditions, condition)
}

func AppErrorStatus(application *Application, error string) {
	condition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  meta.FailedReason,
		Message: error,
	}
	apimeta.SetStatusCondition(&application.Status.Conditions, condition)
}

func AppPodAnalysisCondition(application *Application, result analyzer.Result) bool {
	appPodReadyCondition := apimeta.FindStatusCondition(application.Status.Conditions, PodReady)
	condition := metav1.Condition{
		Type: PodReady, // Could be meta.ReadyCondition, but it would clash with the HR
	}
	if !result.Healthy {

		condition.Message = fmt.Sprintf("%s %s is unhealthy: %v", result.ResourceType, result.ResourceName, result.Errors)
		if appPodReadyCondition == nil || strings.Contains(appPodReadyCondition.Message, PodInitializing) || !strings.Contains(condition.Message, PodInitializing) {
			condition.Reason = ContainersNotReady
			condition.Status = metav1.ConditionFalse
			apimeta.SetStatusCondition(&application.Status.Conditions, condition)
			return true
		}
	}
	return false
}
