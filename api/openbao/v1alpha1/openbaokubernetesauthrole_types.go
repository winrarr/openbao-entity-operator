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

// OpenBaoKubernetesAuthRoleSpec defines the desired state of a role in an
// OpenBao Kubernetes Auth mount.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoKubernetesAuthRole"
// +kubebuilder:validation:XValidation:rule="self.mountPath == oldSelf.mountPath",message="mountPath is immutable; delete and recreate the OpenBaoKubernetesAuthRole"
type OpenBaoKubernetesAuthRoleSpec struct {
	// ConnectionRef selects the OpenBao API connection in the same namespace.
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`

	// MountPath is the OpenBao Kubernetes Auth mount path without the leading
	// auth/ prefix. The mount must already be enabled and configured.
	// +kubebuilder:default="kubernetes"
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

	// BoundServiceAccountNames lists the Kubernetes ServiceAccounts allowed to use the role.
	// OpenBao also accepts * as a wildcard; use it only when the connection's OpenBao
	// policy deliberately permits that boundary.
	// +kubebuilder:validation:MinItems=1
	// +listType=set
	BoundServiceAccountNames []string `json:"boundServiceAccountNames"`

	// BoundServiceAccountNamespaces lists the Kubernetes namespaces allowed to use the role.
	// OpenBao also accepts * as a wildcard; use it only when the connection's OpenBao
	// policy deliberately permits that boundary.
	// +kubebuilder:validation:MinItems=1
	// +listType=set
	BoundServiceAccountNamespaces []string `json:"boundServiceAccountNamespaces"`

	// TokenPolicies is the set of OpenBao ACL policies attached to tokens issued by the role.
	// +listType=set
	// +optional
	TokenPolicies []string `json:"tokenPolicies,omitempty"`

	// TokenTTL is the initial token lifetime. A zero or omitted value uses OpenBao's default.
	// +optional
	TokenTTL *metav1.Duration `json:"tokenTTL,omitempty"`

	// TokenMaxTTL is the maximum token lifetime. A zero or omitted value uses OpenBao's default.
	// +optional
	TokenMaxTTL *metav1.Duration `json:"tokenMaxTTL,omitempty"`

	// TokenPeriod gives issued tokens a fixed renewable period. A zero or omitted value disables it.
	// +optional
	TokenPeriod *metav1.Duration `json:"tokenPeriod,omitempty"`

	// Audience restricts login JWTs to the configured Kubernetes audience.
	// +optional
	Audience string `json:"audience,omitempty"`

	// TokenType selects the type of token issued by the role.
	// +optional
	// +kubebuilder:validation:Enum=service;batch
	TokenType string `json:"tokenType,omitempty"`

	// TokenNumUses limits how many times an issued token may be used. Zero means unlimited.
	// +optional
	TokenNumUses *int64 `json:"tokenNumUses,omitempty"`

	// TokenNoDefaultPolicy prevents OpenBao from adding the default policy to issued tokens.
	// +optional
	TokenNoDefaultPolicy *bool `json:"tokenNoDefaultPolicy,omitempty"`

	// TokenExplicitMaxTTL gives issued tokens an explicit maximum TTL.
	// +optional
	TokenExplicitMaxTTL *metav1.Duration `json:"tokenExplicitMaxTTL,omitempty"`

	// TokenBoundCIDRs restricts issued tokens to these client address ranges.
	// +listType=set
	// +optional
	TokenBoundCIDRs []string `json:"tokenBoundCIDRs,omitempty"`
}

// OpenBaoKubernetesAuthRoleStatus defines the observed state of an OpenBao
// Kubernetes Auth role.
type OpenBaoKubernetesAuthRoleStatus struct {
	// MountPath is the OpenBao auth mount path used for the observed role.
	// +optional
	MountPath string `json:"mountPath,omitempty"`

	// Name is the name returned by OpenBao.
	// +optional
	Name string `json:"name,omitempty"`

	// ConfigHash is the SHA-256 hash of the normalized role configuration returned by OpenBao.
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

// OpenBaoKubernetesAuthRole is the Schema for the openbaokubernetesauthroles API.
type OpenBaoKubernetesAuthRole struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OpenBaoKubernetesAuthRole
	// +required
	Spec OpenBaoKubernetesAuthRoleSpec `json:"spec"`

	// status defines the observed state of OpenBaoKubernetesAuthRole
	// +optional
	Status OpenBaoKubernetesAuthRoleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoKubernetesAuthRoleList contains a list of OpenBaoKubernetesAuthRole.
type OpenBaoKubernetesAuthRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoKubernetesAuthRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoKubernetesAuthRole{}, &OpenBaoKubernetesAuthRoleList{})
		return nil
	})
}
