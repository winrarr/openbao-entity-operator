/*
Copyright 2026 openbao-entity-operator contributors.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// OpenBaoEntitySpec defines the desired state of OpenBaoEntity.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoEntity"
type OpenBaoEntitySpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// CreationPolicy controls how an external entity is acquired.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external entity is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.
	// A zero duration disables periodic checks. When omitted, the operator default is used.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`

	// Metadata is the desired key-value metadata on the OpenBao entity.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// Policies is the desired set of OpenBao ACL policy names.
	// +listType=set
	// +optional
	Policies []string `json:"policies,omitempty"`

	// Disabled prevents tokens associated with the entity from being used.
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

// OpenBaoEntityStatus defines the observed state of OpenBaoEntity.
type OpenBaoEntityStatus struct {
	// ID is the stable OpenBao entity identifier.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the name returned by OpenBao.
	// +optional
	Name string `json:"name,omitempty"`

	// Metadata is the metadata observed on the OpenBao entity.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// Policies is the policy set observed on the OpenBao entity.
	// +listType=set
	// +optional
	Policies []string `json:"policies,omitempty"`

	// Disabled is the disabled state observed on the OpenBao entity.
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the entity.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="OpenBao ID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoEntity is the Schema for the openbaoentities API.
type OpenBaoEntity struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OpenBaoEntity
	// +required
	Spec OpenBaoEntitySpec `json:"spec"`

	// status defines the observed state of OpenBaoEntity
	// +optional
	Status OpenBaoEntityStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoEntityList contains a list of OpenBaoEntity
type OpenBaoEntityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoEntity `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoEntity{}, &OpenBaoEntityList{})
		return nil
	})
}
