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

// OpenBaoPasswordPolicySpec defines a password-generation policy in OpenBao.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoPasswordPolicy"
type OpenBaoPasswordPolicySpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// Rules is the OpenBao password policy document in HCL or JSON syntax.
	// +kubebuilder:validation:MinLength=1
	Rules string `json:"rules"`

	// CreationPolicy controls how an external policy is acquired.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external policy is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.
	// A zero duration disables periodic checks. When omitted, the operator default is used.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`
}

// OpenBaoPasswordPolicyStatus defines the observed state of an OpenBao password policy.
type OpenBaoPasswordPolicyStatus struct {
	// Name is the OpenBao password policy name.
	// +optional
	Name string `json:"name,omitempty"`

	// RulesHash is the SHA-256 hash of the observed policy document.
	// +optional
	RulesHash string `json:"rulesHash,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the policy.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Rules Hash",type=string,JSONPath=`.status.rulesHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoPasswordPolicy is the Schema for the openbaopasswordpolicies API.
type OpenBaoPasswordPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoPasswordPolicySpec   `json:"spec"`
	Status            OpenBaoPasswordPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoPasswordPolicyList contains a list of OpenBaoPasswordPolicy.
type OpenBaoPasswordPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoPasswordPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoPasswordPolicy{}, &OpenBaoPasswordPolicyList{})
		return nil
	})
}
