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

// OpenBaoGroupType identifies the OpenBao identity group membership model.
type OpenBaoGroupType string

const (
	OpenBaoGroupTypeInternal OpenBaoGroupType = "Internal"
	OpenBaoGroupTypeExternal OpenBaoGroupType = "External"
)

// OpenBaoGroupSpec defines the desired state of an OpenBao identity group.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoGroup"
type OpenBaoGroupSpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// Type selects an internal or external OpenBao group. Membership resources
	// are supported for internal groups.
	// +optional
	// +kubebuilder:default=Internal
	// +kubebuilder:validation:Enum=Internal;External
	Type OpenBaoGroupType `json:"type,omitempty"`

	// CreationPolicy controls how an external group is acquired.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external group is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.
	// A zero duration disables periodic checks. When omitted, the operator default is used.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`

	// Metadata is the desired key-value metadata on the OpenBao group.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// Policies is the desired set of OpenBao ACL policy names.
	// +listType=set
	// +optional
	Policies []string `json:"policies,omitempty"`
}

// OpenBaoGroupStatus defines the observed state of OpenBaoGroup.
type OpenBaoGroupStatus struct {
	// ID is the stable OpenBao group identifier.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the name returned by OpenBao.
	// +optional
	Name string `json:"name,omitempty"`

	// Type is the observed OpenBao group type.
	// +optional
	Type OpenBaoGroupType `json:"type,omitempty"`

	// Metadata is the metadata observed on the OpenBao group.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// Policies is the policy set observed on the OpenBao group.
	// +listType=set
	// +optional
	Policies []string `json:"policies,omitempty"`

	// MemberEntityIDs contains all entity IDs currently assigned to the group.
	// +listType=set
	// +optional
	MemberEntityIDs []string `json:"memberEntityIDs,omitempty"`

	// MemberGroupIDs contains all subgroup IDs currently assigned to the group.
	// +listType=set
	// +optional
	MemberGroupIDs []string `json:"memberGroupIDs,omitempty"`

	// ManagedMemberEntityIDs tracks entity memberships claimed by current or
	// previous OpenBaoGroupMembership resources so deletion removes only owned edges.
	// +listType=set
	// +optional
	ManagedMemberEntityIDs []string `json:"managedMemberEntityIDs,omitempty"`

	// ManagedMemberGroupIDs tracks subgroup memberships claimed by current or
	// previous OpenBaoGroupMembership resources so deletion removes only owned edges.
	// +listType=set
	// +optional
	ManagedMemberGroupIDs []string `json:"managedMemberGroupIDs,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the group.
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

// OpenBaoGroup is the Schema for the openbaogroups API.
type OpenBaoGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoGroupSpec   `json:"spec"`
	Status            OpenBaoGroupStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoGroupList contains a list of OpenBaoGroup.
type OpenBaoGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoGroup{}, &OpenBaoGroupList{})
		return nil
	})
}
