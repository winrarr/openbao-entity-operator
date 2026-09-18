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

// OpenBaoOIDCConfigSpec configures the issuer for OpenBao's identity OIDC provider.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCConfig"
type OpenBaoOIDCConfigSpec struct {
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`
	Issuer        string                     `json:"issuer,omitempty"`
}

// OpenBaoOIDCConfigStatus defines the observed OIDC configuration.
type OpenBaoOIDCConfigStatus struct {
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCConfigSpec   `json:"spec"`
	Status            OpenBaoOIDCConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCConfig `json:"items"`
}

// OpenBaoOIDCProviderSpec configures an OpenBao OIDC provider.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCProvider"
type OpenBaoOIDCProviderSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	AllowedClientIDs       []string                   `json:"allowedClientIDs,omitempty"`
	Issuer                 string                     `json:"issuer,omitempty"`
	ScopesSupported        []string                   `json:"scopesSupported,omitempty"`
}

type OpenBaoOIDCProviderStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCProviderSpec   `json:"spec"`
	Status            OpenBaoOIDCProviderStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCProvider `json:"items"`
}

// OpenBaoOIDCClientSpec configures an OpenBao OIDC client.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCClient"
// +kubebuilder:validation:XValidation:rule="self.key == oldSelf.key",message="key is immutable; delete and recreate the OpenBaoOIDCClient"
type OpenBaoOIDCClientSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	AccessTokenTTL         *metav1.Duration           `json:"accessTokenTTL,omitempty"`
	Assignments            []string                   `json:"assignments,omitempty"`
	AuthorizationCode      *bool                      `json:"authorizationCode,omitempty"`
	ClientCredentials      *bool                      `json:"clientCredentials,omitempty"`
	ClientType             string                     `json:"clientType,omitempty"`
	IDTokenTTL             *metav1.Duration           `json:"idTokenTTL,omitempty"`
	Key                    string                     `json:"key,omitempty"`
	RedirectURIs           []string                   `json:"redirectURIs,omitempty"`
}

type OpenBaoOIDCClientStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCClientSpec   `json:"spec"`
	Status            OpenBaoOIDCClientStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCClient `json:"items"`
}

// OpenBaoOIDCKeySpec configures an OpenBao OIDC signing key.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCKey"
type OpenBaoOIDCKeySpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	Algorithm              string                     `json:"algorithm,omitempty"`
	AllowedClientIDs       []string                   `json:"allowedClientIDs,omitempty"`
	RotationPeriod         *metav1.Duration           `json:"rotationPeriod,omitempty"`
	VerificationTTL        *metav1.Duration           `json:"verificationTTL,omitempty"`
}

type OpenBaoOIDCKeyStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCKeySpec   `json:"spec"`
	Status            OpenBaoOIDCKeyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCKey `json:"items"`
}

// OpenBaoOIDCRoleSpec configures an OpenBao identity OIDC role.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCRole"
// +kubebuilder:validation:XValidation:rule="self.key == oldSelf.key",message="key is immutable; delete and recreate the OpenBaoOIDCRole"
type OpenBaoOIDCRoleSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	ClientID               string                     `json:"clientID,omitempty"`
	Key                    string                     `json:"key"`
	Template               string                     `json:"template,omitempty"`
	TTL                    *metav1.Duration           `json:"ttl,omitempty"`
}

type OpenBaoOIDCRoleStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCRoleSpec   `json:"spec"`
	Status            OpenBaoOIDCRoleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCRole `json:"items"`
}

// OpenBaoOIDCScopeSpec configures an OpenBao OIDC scope.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCScope"
type OpenBaoOIDCScopeSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	Description            string                     `json:"description,omitempty"`
	Template               string                     `json:"template,omitempty"`
}

type OpenBaoOIDCScopeStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCScopeSpec   `json:"spec"`
	Status            OpenBaoOIDCScopeStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCScope `json:"items"`
}

// OpenBaoOIDCAssignmentSpec assigns entities and groups to an OIDC assignment.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoOIDCAssignment"
type OpenBaoOIDCAssignmentSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	EntityIDs              []string                   `json:"entityIDs,omitempty"`
	GroupIDs               []string                   `json:"groupIDs,omitempty"`
}

type OpenBaoOIDCAssignmentStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Config Hash",type=string,JSONPath=`.status.configHash`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoOIDCAssignment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoOIDCAssignmentSpec   `json:"spec"`
	Status            OpenBaoOIDCAssignmentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoOIDCAssignmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoOIDCAssignment `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion,
			&OpenBaoOIDCConfig{}, &OpenBaoOIDCConfigList{},
			&OpenBaoOIDCProvider{}, &OpenBaoOIDCProviderList{},
			&OpenBaoOIDCClient{}, &OpenBaoOIDCClientList{},
			&OpenBaoOIDCKey{}, &OpenBaoOIDCKeyList{},
			&OpenBaoOIDCRole{}, &OpenBaoOIDCRoleList{},
			&OpenBaoOIDCScope{}, &OpenBaoOIDCScopeList{},
			&OpenBaoOIDCAssignment{}, &OpenBaoOIDCAssignmentList{},
		)
		return nil
	})
}
