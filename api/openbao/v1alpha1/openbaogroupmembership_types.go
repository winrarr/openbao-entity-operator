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

// OpenBaoGroupMembershipSpec defines one explicit membership edge.
// +kubebuilder:validation:XValidation:rule="self.groupRef == oldSelf.groupRef",message="groupRef is immutable; delete and recreate the OpenBaoGroupMembership"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.entityRef) || (has(self.entityRef) && self.entityRef == oldSelf.entityRef)",message="entityRef is immutable; delete and recreate the OpenBaoGroupMembership"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.memberGroupRef) || (has(self.memberGroupRef) && self.memberGroupRef == oldSelf.memberGroupRef)",message="memberGroupRef is immutable; delete and recreate the OpenBaoGroupMembership"
// +kubebuilder:validation:XValidation:rule="has(self.entityRef) != has(self.memberGroupRef)",message="exactly one of entityRef or memberGroupRef must be set"
type OpenBaoGroupMembershipSpec struct {
	// GroupRef selects the parent OpenBaoGroup in the same namespace.
	GroupRef OpenBaoGroupReference `json:"groupRef"`

	// EntityRef selects an OpenBaoEntity to add to the parent group.
	// +optional
	EntityRef *OpenBaoEntityReference `json:"entityRef,omitempty"`

	// MemberGroupRef selects an OpenBaoGroup to add as a subgroup of the parent group.
	// +optional
	MemberGroupRef *OpenBaoGroupReference `json:"memberGroupRef,omitempty"`
}

// OpenBaoGroupMembershipStatus defines the observed state of one membership edge.
type OpenBaoGroupMembershipStatus struct {
	// GroupID is the stable parent OpenBao group identifier.
	// +optional
	GroupID string `json:"groupID,omitempty"`

	// MemberID is the stable entity or subgroup identifier.
	// +optional
	MemberID string `json:"memberID,omitempty"`

	// MemberType identifies whether MemberID is an entity or subgroup.
	// +optional
	// +kubebuilder:validation:Enum=Entity;Group
	MemberType string `json:"memberType,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the membership.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Group ID",type=string,JSONPath=`.status.groupID`
// +kubebuilder:printcolumn:name="Member ID",type=string,JSONPath=`.status.memberID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoGroupMembership is the Schema for the openbaogroupmemberships API.
type OpenBaoGroupMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoGroupMembershipSpec   `json:"spec"`
	Status            OpenBaoGroupMembershipStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoGroupMembershipList contains a list of OpenBaoGroupMembership.
type OpenBaoGroupMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoGroupMembership `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoGroupMembership{}, &OpenBaoGroupMembershipList{})
		return nil
	})
}
