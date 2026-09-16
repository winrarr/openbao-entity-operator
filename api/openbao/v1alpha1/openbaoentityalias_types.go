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

// OpenBaoEntityAliasSpec defines the desired state of an OpenBao entity alias.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoEntityAlias"
// +kubebuilder:validation:XValidation:rule="self.entityRef == oldSelf.entityRef",message="entityRef is immutable; delete and recreate the OpenBaoEntityAlias"
// +kubebuilder:validation:XValidation:rule="self.mountAccessor == oldSelf.mountAccessor",message="mountAccessor is immutable; delete and recreate the OpenBaoEntityAlias"
// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="name is immutable; delete and recreate the OpenBaoEntityAlias"
type OpenBaoEntityAliasSpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// EntityRef selects the OpenBaoEntity receiving this alias in the same namespace.
	EntityRef OpenBaoEntityReference `json:"entityRef"`

	// MountAccessor is the OpenBao auth-method mount accessor for this alias.
	// +kubebuilder:validation:MinLength=1
	MountAccessor string `json:"mountAccessor"`

	// Name is the name presented by the auth method for this alias.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// CreationPolicy controls how an external alias is acquired.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external alias is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.
	// A zero duration disables periodic checks. When omitted, the operator default is used.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`
}

// OpenBaoEntityAliasStatus defines the observed state of OpenBaoEntityAlias.
type OpenBaoEntityAliasStatus struct {
	// ID is the stable OpenBao entity alias identifier.
	// +optional
	ID string `json:"id,omitempty"`

	// CanonicalID is the OpenBao entity identifier receiving this alias.
	// +optional
	CanonicalID string `json:"canonicalID,omitempty"`

	// Name is the alias name returned by OpenBao.
	// +optional
	Name string `json:"name,omitempty"`

	// MountAccessor is the auth-method mount accessor returned by OpenBao.
	// +optional
	MountAccessor string `json:"mountAccessor,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the alias.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="OpenBao ID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="Entity ID",type=string,JSONPath=`.status.canonicalID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoEntityAlias is the Schema for the openbaoentityaliases API.
type OpenBaoEntityAlias struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OpenBaoEntityAlias
	// +required
	Spec OpenBaoEntityAliasSpec `json:"spec"`

	// status defines the observed state of OpenBaoEntityAlias
	// +optional
	Status OpenBaoEntityAliasStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoEntityAliasList contains a list of OpenBaoEntityAlias.
type OpenBaoEntityAliasList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoEntityAlias `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoEntityAlias{}, &OpenBaoEntityAliasList{})
		return nil
	})
}
