/*
Copyright 2026.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VibeAppSpec defines the desired state of VibeApp
type VibeAppSpec struct {
	// Image is the container image for the application
	// +required
	Image string `json:"image"`

	// Replicas is the desired number of pod replicas
	// Minimum value enforced to be 2 (either by mutating webhook or controller self-healing)
	// +required
	// +kubebuilder:validation:Minimum=2
	Replicas int32 `json:"replicas"`

	// HealthCheckPath is the HTTP endpoint for liveness/readiness probes
	// +required
	HealthCheckPath string `json:"healthCheckPath"`

	// StoragePath is the host path to mount into the container
	// +required
	StoragePath string `json:"storagePath"`

	// Port is the container port that the Service will expose
	// +required
	Port int32 `json:"port"`
}

// VibeAppStatus defines the observed state of VibeApp.
type VibeAppStatus struct {
	// conditions represent the current state of the VibeApp resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// VibeApp is the Schema for the vibeapps API
type VibeApp struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of VibeApp
	// +required
	Spec VibeAppSpec `json:"spec"`

	// status defines the observed state of VibeApp
	// +optional
	Status VibeAppStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// VibeAppList contains a list of VibeApp
type VibeAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []VibeApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VibeApp{}, &VibeAppList{})
}
