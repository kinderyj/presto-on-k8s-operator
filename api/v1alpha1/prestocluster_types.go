/*
Copyright 2020 yujunwang.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PrestoClusterSpec defines the desired state of PrestoCluster
type PrestoClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name              string            `json:"name"`
	Image             string            `json:"image"`
	Workers           *int32            `json:"workers"`
	CoordinatorConfig CoordinatorConfig `json:"coordinatorConfig"`
	WorkerConfig      WorkerConfig      `json:"workerConfig"`
	CatalogConfig     map[string]string `json:"catalogConfig,omitempty"`
	Coresite          map[string]string `json:"coresite,omitempty"`
}

// CoordinatorConfig defines the coordinator config
type CoordinatorConfig struct {
	// Compute resources required by each Coordinator container.
	// If omitted, a default value will be used.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	EtcConfig map[string]string           `json:"etcConfig,omitempty"`
}

// WorkerConfig defines the worker config.
type WorkerConfig struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	EtcConfig map[string]string           `json:"etcConfig,omitempty"`
}

// PrestoClusterStatus defines the inspected state of PrestoCluster
type PrestoClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define inspected state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AvailableWorkers int32 `json:"availableWorkers"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=pcs

// PrestoCluster is the Schema for the prestoclusters API
type PrestoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrestoClusterSpec   `json:"spec,omitempty"`
	Status PrestoClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PrestoClusterList contains a list of PrestoCluster
type PrestoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrestoCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrestoCluster{}, &PrestoClusterList{})
}
