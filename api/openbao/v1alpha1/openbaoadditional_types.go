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

// OpenBaoPersonaSpec configures a durable OpenBao identity persona.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoPersona"
// +kubebuilder:validation:XValidation:rule="self.mountAccessor == oldSelf.mountAccessor",message="mountAccessor is immutable; delete and recreate the OpenBaoPersona"
type OpenBaoPersonaSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Name                   string                     `json:"name"`
	EntityID               string                     `json:"entityID"`
	MountAccessor          string                     `json:"mountAccessor"`
	Metadata               map[string]string          `json:"metadata,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoPersonaStatus struct {
	ID                 string             `json:"id,omitempty"`
	Name               string             `json:"name,omitempty"`
	EntityID           string             `json:"entityID,omitempty"`
	MountAccessor      string             `json:"mountAccessor,omitempty"`
	Metadata           map[string]string  `json:"metadata,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="OpenBao ID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoPersona struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoPersonaSpec   `json:"spec"`
	Status            OpenBaoPersonaStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoPersonaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoPersona `json:"items"`
}

// OpenBaoMFALoginEnforcementSpec configures which MFA methods are enforced.
type OpenBaoMFALoginEnforcementSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Name                   string                     `json:"name"`
	AuthMethodAccessors    []string                   `json:"authMethodAccessors,omitempty"`
	AuthMethodTypes        []string                   `json:"authMethodTypes,omitempty"`
	IdentityEntityIDs      []string                   `json:"identityEntityIDs,omitempty"`
	IdentityGroupIDs       []string                   `json:"identityGroupIDs,omitempty"`
	MFAMethodIDs           []string                   `json:"mfaMethodIDs"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoMFALoginEnforcementStatus struct {
	Name               string             `json:"name,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoMFALoginEnforcement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoMFALoginEnforcementSpec   `json:"spec"`
	Status            OpenBaoMFALoginEnforcementStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoMFALoginEnforcementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoMFALoginEnforcement `json:"items"`
}

// OpenBaoMFAMethodType identifies the native OpenBao MFA provider.
type OpenBaoMFAMethodType string

const (
	OpenBaoMFAMethodTypeDuo    OpenBaoMFAMethodType = "duo"
	OpenBaoMFAMethodTypeOkta   OpenBaoMFAMethodType = "okta"
	OpenBaoMFAMethodTypePingID OpenBaoMFAMethodType = "pingid"
	OpenBaoMFAMethodTypeTOTP   OpenBaoMFAMethodType = "totp"
)

// OpenBaoMFAMethodSpec configures an MFA provider. Secret-bearing provider
// values are read from same-namespace Secrets and are never written to status.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the OpenBaoMFAMethod"
// +kubebuilder:validation:XValidation:rule="self.type == oldSelf.type",message="type is immutable; delete and recreate the OpenBaoMFAMethod"
// +kubebuilder:validation:XValidation:rule="self.methodID == oldSelf.methodID",message="methodID is immutable; delete and recreate the OpenBaoMFAMethod"
type OpenBaoMFAMethodSpec struct {
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`
	// +kubebuilder:validation:Enum=duo;okta;pingid;totp
	Type           OpenBaoMFAMethodType `json:"type"`
	MethodID       string               `json:"methodID"`
	MethodName     string               `json:"methodName"`
	UsernameFormat string               `json:"usernameFormat,omitempty"`

	// Duo configuration.
	APIHostname    string              `json:"apiHostname,omitempty"`
	IntegrationKey string              `json:"integrationKey,omitempty"`
	SecretKeyRef   *SecretKeyReference `json:"secretKeyRef,omitempty"`
	PushInfo       string              `json:"pushInfo,omitempty"`
	UsePasscode    *bool               `json:"usePasscode,omitempty"`

	// Okta configuration.
	APITokenRef  *SecretKeyReference `json:"apiTokenRef,omitempty"`
	BaseURL      string              `json:"baseURL,omitempty"`
	OrgName      string              `json:"orgName,omitempty"`
	PrimaryEmail *bool               `json:"primaryEmail,omitempty"`
	Production   *bool               `json:"production,omitempty"`

	// PingID configuration.
	SettingsFileRef *SecretKeyReference `json:"settingsFileRef,omitempty"`

	// TOTP configuration. TOTP secret generation is intentionally not part of
	// this resource; these fields configure only the durable method settings.
	Algorithm             string `json:"algorithm,omitempty"`
	Digits                *int32 `json:"digits,omitempty"`
	Issuer                string `json:"issuer,omitempty"`
	KeySize               *int32 `json:"keySize,omitempty"`
	MaxValidationAttempts *int32 `json:"maxValidationAttempts,omitempty"`
	Period                *int32 `json:"period,omitempty"`
	QRSize                *int32 `json:"qrSize,omitempty"`
	Skew                  *int32 `json:"skew,omitempty"`

	CreationPolicy         CreationPolicy   `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy   `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoMFAMethodStatus struct {
	MethodID           string             `json:"methodID,omitempty"`
	MethodName         string             `json:"methodName,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	SecretHash         string             `json:"secretHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Method ID",type=string,JSONPath=`.status.methodID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoMFAMethod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoMFAMethodSpec   `json:"spec"`
	Status            OpenBaoMFAMethodStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoMFAMethodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoMFAMethod `json:"items"`
}

// OpenBaoCORSConfigurationSpec configures the singleton OpenBao CORS policy.
type OpenBaoCORSConfigurationSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Enable                 *bool                      `json:"enable,omitempty"`
	AllowCredentials       *bool                      `json:"allowCredentials,omitempty"`
	AllowedHeaders         []string                   `json:"allowedHeaders,omitempty"`
	AllowedOrigins         []string                   `json:"allowedOrigins,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoCORSConfigurationStatus struct {
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoCORSConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoCORSConfigurationSpec   `json:"spec"`
	Status            OpenBaoCORSConfigurationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoCORSConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoCORSConfiguration `json:"items"`
}

// OpenBaoAuditRequestHeaderSpec configures one audit request header.
type OpenBaoAuditRequestHeaderSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Header                 string                     `json:"header"`
	HMAC                   *bool                      `json:"hmac,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoAuditRequestHeaderStatus struct {
	Header             string             `json:"header,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Header",type=string,JSONPath=`.status.header`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoAuditRequestHeader struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoAuditRequestHeaderSpec   `json:"spec"`
	Status            OpenBaoAuditRequestHeaderStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoAuditRequestHeaderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoAuditRequestHeader `json:"items"`
}

// OpenBaoUIHeaderSpec configures one OpenBao UI response header.
type OpenBaoUIHeaderSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Header                 string                     `json:"header"`
	Values                 []string                   `json:"values,omitempty"`
	Multivalue             *bool                      `json:"multivalue,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoUIHeaderStatus struct {
	Header             string             `json:"header,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Header",type=string,JSONPath=`.status.header`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoUIHeader struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoUIHeaderSpec   `json:"spec"`
	Status            OpenBaoUIHeaderStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoUIHeaderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoUIHeader `json:"items"`
}

// OpenBaoRateLimitQuotaConfigurationSpec configures global quota behavior.
type OpenBaoRateLimitQuotaConfigurationSpec struct {
	ConnectionRef                  OpenBaoConnectionReference `json:"connectionRef"`
	EnableRateLimitAuditLogging    *bool                      `json:"enableRateLimitAuditLogging,omitempty"`
	EnableRateLimitResponseHeaders *bool                      `json:"enableRateLimitResponseHeaders,omitempty"`
	RateLimitExemptPaths           []string                   `json:"rateLimitExemptPaths,omitempty"`
	CreationPolicy                 CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy                 DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval         *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoRateLimitQuotaConfigurationStatus struct {
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoRateLimitQuotaConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoRateLimitQuotaConfigurationSpec   `json:"spec"`
	Status            OpenBaoRateLimitQuotaConfigurationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoRateLimitQuotaConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoRateLimitQuotaConfiguration `json:"items"`
}

// OpenBaoLoggerSpec configures OpenBao logger verbosity. An empty Name targets
// the global logger; a non-empty Name targets one named subsystem logger.
type OpenBaoLoggerSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Name                   string                     `json:"name,omitempty"`
	Level                  string                     `json:"level"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoLoggerStatus struct {
	Name               string             `json:"name,omitempty"`
	Level              string             `json:"level,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.status.name`
// +kubebuilder:printcolumn:name="Level",type=string,JSONPath=`.status.level`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoLogger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoLoggerSpec   `json:"spec"`
	Status            OpenBaoLoggerStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoLoggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoLogger `json:"items"`
}

// OpenBaoRotationConfigurationSpec configures automatic encryption-key
// rotation. It does not perform an immediate rotation.
type OpenBaoRotationConfigurationSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Enabled                *bool                      `json:"enabled,omitempty"`
	Interval               *metav1.Duration           `json:"interval,omitempty"`
	MaxOperations          *int64                     `json:"maxOperations,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}

type OpenBaoRotationConfigurationStatus struct {
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoEncryptionKeyConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoRotationConfigurationSpec   `json:"spec"`
	Status            OpenBaoRotationConfigurationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoEncryptionKeyConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoEncryptionKeyConfiguration `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoKeyringRotationConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoRotationConfigurationSpec   `json:"spec"`
	Status            OpenBaoRotationConfigurationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoKeyringRotationConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoKeyringRotationConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion,
			&OpenBaoPersona{}, &OpenBaoPersonaList{},
			&OpenBaoMFALoginEnforcement{}, &OpenBaoMFALoginEnforcementList{},
			&OpenBaoMFAMethod{}, &OpenBaoMFAMethodList{},
			&OpenBaoCORSConfiguration{}, &OpenBaoCORSConfigurationList{},
			&OpenBaoAuditRequestHeader{}, &OpenBaoAuditRequestHeaderList{},
			&OpenBaoUIHeader{}, &OpenBaoUIHeaderList{},
			&OpenBaoRateLimitQuotaConfiguration{}, &OpenBaoRateLimitQuotaConfigurationList{},
			&OpenBaoLogger{}, &OpenBaoLoggerList{},
			&OpenBaoEncryptionKeyConfiguration{}, &OpenBaoEncryptionKeyConfigurationList{},
			&OpenBaoKeyringRotationConfiguration{}, &OpenBaoKeyringRotationConfigurationList{},
		)
		return nil
	})
}
