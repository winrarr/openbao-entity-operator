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

// OpenBaoGroupAliasSpec defines an alias that maps an authentication mount
// identity to an OpenBao identity group.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoGroupAlias"
// +kubebuilder:validation:XValidation:rule="self.groupRef == oldSelf.groupRef",message="groupRef is immutable; delete and recreate the OpenBaoGroupAlias"
// +kubebuilder:validation:XValidation:rule="self.mountAccessor == oldSelf.mountAccessor",message="mountAccessor is immutable; delete and recreate the OpenBaoGroupAlias"
// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="name is immutable; delete and recreate the OpenBaoGroupAlias"
type OpenBaoGroupAliasSpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// GroupRef selects the OpenBaoGroup receiving this alias in the same namespace.
	GroupRef OpenBaoGroupReference `json:"groupRef"`

	// MountAccessor is the OpenBao auth-method mount accessor for this alias.
	// +kubebuilder:validation:MinLength=1
	MountAccessor string `json:"mountAccessor"`

	// Name is the group name presented by the authentication method.
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

// OpenBaoGroupAliasStatus defines the observed state of an OpenBao group alias.
type OpenBaoGroupAliasStatus struct {
	// ID is the stable OpenBao group alias identifier.
	// +optional
	ID string `json:"id,omitempty"`

	// CanonicalID is the OpenBao group identifier receiving this alias.
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
// +kubebuilder:printcolumn:name="Group ID",type=string,JSONPath=`.status.canonicalID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoGroupAlias is the Schema for the openbaogroupaliases API.
type OpenBaoGroupAlias struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoGroupAliasSpec   `json:"spec"`
	Status            OpenBaoGroupAliasStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoGroupAliasList contains a list of OpenBaoGroupAlias.
type OpenBaoGroupAliasList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoGroupAlias `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoGroupAlias{}, &OpenBaoGroupAliasList{})
		return nil
	})
}
