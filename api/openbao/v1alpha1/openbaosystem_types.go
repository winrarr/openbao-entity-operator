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

// OpenBaoMountSpec is shared by auth-method and secret-engine resources.
type OpenBaoMountSpec struct {
	ConnectionRef             OpenBaoConnectionReference `json:"connectionRef"`
	Path                      string                     `json:"path"`
	Type                      string                     `json:"type"`
	Description               string                     `json:"description,omitempty"`
	CreationPolicy            CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy            DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval    *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
	Local                     *bool                      `json:"local,omitempty"`
	SealWrap                  *bool                      `json:"sealWrap,omitempty"`
	ExternalEntropyAccess     *bool                      `json:"externalEntropyAccess,omitempty"`
	PluginName                string                     `json:"pluginName,omitempty"`
	PluginVersion             string                     `json:"pluginVersion,omitempty"`
	Options                   map[string]string          `json:"options,omitempty"`
	Config                    map[string]string          `json:"config,omitempty"`
	DefaultLeaseTTL           *metav1.Duration           `json:"defaultLeaseTTL,omitempty"`
	MaxLeaseTTL               *metav1.Duration           `json:"maxLeaseTTL,omitempty"`
	ListingVisibility         string                     `json:"listingVisibility,omitempty"`
	TokenType                 string                     `json:"tokenType,omitempty"`
	PassthroughRequestHeaders []string                   `json:"passthroughRequestHeaders,omitempty"`
	AllowedResponseHeaders    []string                   `json:"allowedResponseHeaders,omitempty"`
}

type OpenBaoMountStatus struct {
	Path               string             `json:"path,omitempty"`
	Accessor           string             `json:"accessor,omitempty"`
	Type               string             `json:"type,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// OpenBaoAuthMethodSpec enables and tunes an OpenBao auth method.
type OpenBaoAuthMethodSpec struct {
	OpenBaoMountSpec `json:",inline"`
}
type OpenBaoAuthMethodStatus struct {
	OpenBaoMountStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.status.path`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoAuthMethod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoAuthMethodSpec   `json:"spec"`
	Status            OpenBaoAuthMethodStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoAuthMethodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoAuthMethod `json:"items"`
}

// OpenBaoSecretEngineSpec enables and tunes an OpenBao secret engine.
type OpenBaoSecretEngineSpec struct {
	OpenBaoMountSpec `json:",inline"`
}
type OpenBaoSecretEngineStatus struct {
	OpenBaoMountStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.status.path`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoSecretEngine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoSecretEngineSpec   `json:"spec"`
	Status            OpenBaoSecretEngineStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoSecretEngineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoSecretEngine `json:"items"`
}

// OpenBaoNamespaceSpec manages a child OpenBao namespace inside the connection's namespace.
type OpenBaoNamespaceSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Path                   string                     `json:"path"`
	CustomMetadata         map[string]string          `json:"customMetadata,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}
type OpenBaoNamespaceStatus struct {
	Path               string             `json:"path,omitempty"`
	ID                 string             `json:"id,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.status.path`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoNamespaceSpec   `json:"spec"`
	Status            OpenBaoNamespaceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoNamespace `json:"items"`
}

// OpenBaoAuditDeviceSpec configures an OpenBao audit device.
type OpenBaoAuditDeviceSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Path                   string                     `json:"path"`
	Type                   string                     `json:"type"`
	Description            string                     `json:"description,omitempty"`
	Local                  *bool                      `json:"local,omitempty"`
	Options                map[string]string          `json:"options,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}
type OpenBaoAuditDeviceStatus struct {
	Path               string             `json:"path,omitempty"`
	Type               string             `json:"type,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.status.path`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoAuditDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoAuditDeviceSpec   `json:"spec"`
	Status            OpenBaoAuditDeviceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoAuditDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoAuditDevice `json:"items"`
}

// OpenBaoRateLimitQuotaSpec configures a rate-limit quota.
type OpenBaoRateLimitQuotaSpec struct {
	ConnectionRef OpenBaoConnectionReference `json:"connectionRef"`
	Type          string                     `json:"type"`
	Path          string                     `json:"path,omitempty"`
	Role          string                     `json:"role,omitempty"`
	// Rate is the positive request rate. It is encoded as a string to preserve
	// exact decimal values across Kubernetes clients and CRD serializers.
	Rate                   string           `json:"rate,omitempty"`
	Interval               *metav1.Duration `json:"interval,omitempty"`
	BlockInterval          *metav1.Duration `json:"blockInterval,omitempty"`
	Inheritable            *bool            `json:"inheritable,omitempty"`
	CreationPolicy         CreationPolicy   `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy   `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`
}
type OpenBaoRateLimitQuotaStatus struct {
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
type OpenBaoRateLimitQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoRateLimitQuotaSpec   `json:"spec"`
	Status            OpenBaoRateLimitQuotaStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoRateLimitQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoRateLimitQuota `json:"items"`
}

// OpenBaoWorkflowSpec manages a stored OpenBao workflow definition.
type OpenBaoWorkflowSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Path                   string                     `json:"path"`
	Workflow               string                     `json:"workflow"`
	Description            string                     `json:"description,omitempty"`
	AllowUnauthenticated   *bool                      `json:"allowUnauthenticated,omitempty"`
	CAS                    *int64                     `json:"cas,omitempty"`
	CASRequired            *bool                      `json:"casRequired,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}
type OpenBaoWorkflowStatus struct {
	Path               string             `json:"path,omitempty"`
	Version            int64              `json:"version,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.status.path`
// +kubebuilder:printcolumn:name="Version",type=integer,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoWorkflowSpec   `json:"spec"`
	Status            OpenBaoWorkflowStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoWorkflow `json:"items"`
}

// OpenBaoPluginSpec registers a plugin that is already present in OpenBao's plugin directory.
type OpenBaoPluginSpec struct {
	ConnectionRef          OpenBaoConnectionReference `json:"connectionRef"`
	Name                   string                     `json:"name"`
	Type                   string                     `json:"type"`
	Command                string                     `json:"command,omitempty"`
	Args                   []string                   `json:"args,omitempty"`
	Env                    []string                   `json:"env,omitempty"`
	SHA256                 string                     `json:"sha256,omitempty"`
	Version                string                     `json:"version,omitempty"`
	OCI                    *bool                      `json:"oci,omitempty"`
	CreationPolicy         CreationPolicy             `json:"creationPolicy,omitempty"`
	DeletionPolicy         DeletionPolicy             `json:"deletionPolicy,omitempty"`
	DriftDetectionInterval *metav1.Duration           `json:"driftDetectionInterval,omitempty"`
}
type OpenBaoPluginStatus struct {
	Name               string             `json:"name,omitempty"`
	Type               string             `json:"type,omitempty"`
	ConfigHash         string             `json:"configHash,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenBaoPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              OpenBaoPluginSpec   `json:"spec"`
	Status            OpenBaoPluginStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
type OpenBaoPluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoPlugin `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion,
			&OpenBaoAuthMethod{}, &OpenBaoAuthMethodList{},
			&OpenBaoSecretEngine{}, &OpenBaoSecretEngineList{},
			&OpenBaoNamespace{}, &OpenBaoNamespaceList{},
			&OpenBaoAuditDevice{}, &OpenBaoAuditDeviceList{},
			&OpenBaoRateLimitQuota{}, &OpenBaoRateLimitQuotaList{},
			&OpenBaoWorkflow{}, &OpenBaoWorkflowList{},
			&OpenBaoPlugin{}, &OpenBaoPluginList{},
		)
		return nil
	})
}
