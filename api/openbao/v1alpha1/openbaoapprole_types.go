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

// OpenBaoAppRoleSpec defines a role in an already enabled OpenBao AppRole mount.
// The resource manages role configuration only; Secret IDs are issued and
// delivered by an external credential workflow.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoAppRole"
// +kubebuilder:validation:XValidation:rule="self.mountPath == oldSelf.mountPath",message="mountPath is immutable; delete and recreate the OpenBaoAppRole"
type OpenBaoAppRoleSpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// MountPath is the AppRole auth mount path without the leading auth/ prefix.
	// +kubebuilder:default=approle
	// +kubebuilder:validation:Pattern=`^[^/[:space:]]+([/][^/[:space:]]+)*$`
	MountPath string `json:"mountPath,omitempty"`

	// CreationPolicy controls how an external role is acquired.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external role is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.
	// A zero duration disables periodic checks. When omitted, the operator default is used.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`

	// BindSecretID requires a Secret ID for login.
	// +optional
	BindSecretID *bool `json:"bindSecretID,omitempty"`

	// LocalSecretIDs stores Secret IDs locally to this AppRole mount.
	// +optional
	LocalSecretIDs *bool `json:"localSecretIDs,omitempty"`

	// SecretIDBoundCIDRs restricts Secret ID use to these client address ranges.
	// +listType=set
	// +optional
	SecretIDBoundCIDRs []string `json:"secretIDBoundCIDRs,omitempty"`

	// SecretIDNumUses limits how many times a Secret ID may be used. Zero means unlimited.
	// +optional
	SecretIDNumUses *int64 `json:"secretIDNumUses,omitempty"`

	// SecretIDTTL is the lifetime of issued Secret IDs.
	// +optional
	SecretIDTTL *metav1.Duration `json:"secretIDTTL,omitempty"`

	// TokenBoundCIDRs restricts issued tokens to these client address ranges.
	// +listType=set
	// +optional
	TokenBoundCIDRs []string `json:"tokenBoundCIDRs,omitempty"`

	// TokenExplicitMaxTTL gives issued tokens an explicit maximum TTL.
	// +optional
	TokenExplicitMaxTTL *metav1.Duration `json:"tokenExplicitMaxTTL,omitempty"`

	// TokenMaxTTL is the maximum lifetime of issued tokens.
	// +optional
	TokenMaxTTL *metav1.Duration `json:"tokenMaxTTL,omitempty"`

	// TokenNoDefaultPolicy prevents OpenBao from adding the default policy.
	// +optional
	TokenNoDefaultPolicy *bool `json:"tokenNoDefaultPolicy,omitempty"`

	// TokenNumUses limits how many times an issued token may be used. Zero means unlimited.
	// +optional
	TokenNumUses *int64 `json:"tokenNumUses,omitempty"`

	// TokenPeriod gives issued tokens a fixed renewable period.
	// +optional
	TokenPeriod *metav1.Duration `json:"tokenPeriod,omitempty"`

	// TokenPolicies is the set of ACL policies attached to issued tokens.
	// +listType=set
	// +optional
	TokenPolicies []string `json:"tokenPolicies,omitempty"`

	// TokenTTL is the initial lifetime of issued tokens.
	// +optional
	TokenTTL *metav1.Duration `json:"tokenTTL,omitempty"`

	// TokenType selects the type of token issued by the role.
	// +optional
	// +kubebuilder:validation:Enum=service;batch
	TokenType string `json:"tokenType,omitempty"`
}

// OpenBaoAppRoleStatus defines the observed state of an OpenBao AppRole.
type OpenBaoAppRoleStatus struct {
	// MountPath is the OpenBao auth mount path used for the observed role.
	// +optional
	MountPath string `json:"mountPath,omitempty"`

	// Name is the OpenBao role name.
	// +optional
	Name string `json:"name,omitempty"`

	// ConfigHash is the SHA-256 hash of the normalized role configuration.
	// +optional
	ConfigHash string `json:"configHash,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the role.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Mount",type=string,JSONPath=`.status.mountPath`
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoAppRole is the Schema for the openbaoapproles API.
type OpenBaoAppRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoAppRoleSpec   `json:"spec"`
	Status            OpenBaoAppRoleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoAppRoleList contains a list of OpenBaoAppRole.
type OpenBaoAppRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoAppRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoAppRole{}, &OpenBaoAppRoleList{})
		return nil
	})
}
