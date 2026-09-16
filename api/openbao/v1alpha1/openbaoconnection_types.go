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

// OpenBaoConnectionSpec defines the desired state of OpenBaoConnection.
// +kubebuilder:validation:XValidation:rule="(has(self.tokenSecretRef) && !has(self.kubernetesAuth) && !has(self.appRole)) || (!has(self.tokenSecretRef) && has(self.kubernetesAuth) && !has(self.appRole)) || (!has(self.tokenSecretRef) && !has(self.kubernetesAuth) && has(self.appRole))",message="exactly one of tokenSecretRef, kubernetesAuth, or appRole must be configured"
type OpenBaoConnectionSpec struct {
	// Address is the OpenBao API address without the /v1 API prefix.
	// +kubebuilder:validation:Pattern=`^https?://`
	// +kubebuilder:validation:MinLength=1
	Address string `json:"address"`

	// Namespace is an optional absolute or relative OpenBao namespace path.
	// An empty value targets the root namespace. The value is sent as the
	// X-Vault-Namespace request header.
	// +optional
	// +kubebuilder:validation:Pattern=`^$|^[^/[:space:]]+([/][^/[:space:]]+)*$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="namespace is immutable; delete and recreate the OpenBaoConnection"
	Namespace string `json:"namespace,omitempty"`

	// TokenSecretRef references a same-namespace Secret containing an OpenBao token.
	// Exactly one of TokenSecretRef, KubernetesAuth, and AppRole must be configured.
	// +optional
	TokenSecretRef *SecretKeyReference `json:"tokenSecretRef,omitempty"`

	// KubernetesAuth logs the operator into OpenBao with its projected Kubernetes
	// ServiceAccount token. The Kubernetes auth method must already be enabled
	// and configured at the selected mount path.
	// Exactly one of TokenSecretRef, KubernetesAuth, and AppRole must be configured.
	// +optional
	KubernetesAuth *KubernetesAuthSpec `json:"kubernetesAuth,omitempty"`

	// AppRole logs the operator into OpenBao with an AppRole role ID and Secret ID
	// read from same-namespace Secrets. The AppRole auth method and role must
	// already be configured in OpenBao.
	// Exactly one of TokenSecretRef, KubernetesAuth, and AppRole must be configured.
	// +optional
	AppRole *AppRoleAuthSpec `json:"appRole,omitempty"`

	// CABundleSecretRef optionally references a same-namespace Secret containing a PEM CA bundle.
	// The key defaults to ca.crt when omitted.
	// +optional
	CABundleSecretRef *SecretKeyReference `json:"caBundleSecretRef,omitempty"`

	// RequestTimeout bounds each request made to OpenBao.
	// +optional
	// +kubebuilder:default="30s"
	RequestTimeout *metav1.Duration `json:"requestTimeout,omitempty"`
}

// KubernetesAuthSpec defines how an OpenBaoConnection uses the Kubernetes auth
// method. The operator reads its projected ServiceAccount JWT from the standard
// in-cluster token path and never stores that JWT in Kubernetes status.
type KubernetesAuthSpec struct {
	// MountPath is the OpenBao auth mount path without the leading auth/ prefix.
	// +optional
	// +kubebuilder:default="kubernetes"
	// +kubebuilder:validation:Pattern=`^[^/[:space:]]+([/][^/[:space:]]+)*$`
	MountPath string `json:"mountPath,omitempty"`

	// Role is the role configured in the OpenBao Kubernetes auth method.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Pattern=`^[^[:space:]]+$`
	Role string `json:"role"`
}

// AppRoleAuthSpec defines how an OpenBaoConnection uses the AppRole auth
// method. Both credential references are reread when a new login is needed.
type AppRoleAuthSpec struct {
	// MountPath is the auth mount path without the leading auth/ prefix.
	// +optional
	// +kubebuilder:default="approle"
	// +kubebuilder:validation:Pattern=`^[^/[:space:]]+([/][^/[:space:]]+)*$`
	MountPath string `json:"mountPath,omitempty"`

	// RoleIDSecretRef references a same-namespace Secret containing the AppRole
	// role ID. The key defaults to role-id.
	RoleIDSecretRef SecretKeyReference `json:"roleIDSecretRef"`

	// SecretIDSecretRef references a same-namespace Secret containing the AppRole
	// Secret ID. The key defaults to secret-id.
	SecretIDSecretRef SecretKeyReference `json:"secretIDSecretRef"`
}

// OpenBaoConnectionStatus defines the observed state of OpenBaoConnection.
type OpenBaoConnectionStatus struct {
	// ObservedGeneration is the most recent generation reflected in status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Version is the OpenBao server version observed during the last root-namespace health check.
	// It is empty for namespace-scoped connections because OpenBao does not expose sys/health in a namespace.
	// +optional
	Version string `json:"version,omitempty"`

	// Initialized reports whether OpenBao has been initialized according to the last root-namespace health check.
	// It is not populated for namespace-scoped connections.
	// +optional
	Initialized bool `json:"initialized,omitempty"`

	// Sealed reports whether OpenBao was sealed during the last root-namespace health check.
	// It is not populated for namespace-scoped connections.
	// +optional
	Sealed bool `json:"sealed,omitempty"`

	// Standby reports whether this server is a standby node according to the last root-namespace health check.
	// It is not populated for namespace-scoped connections.
	// +optional
	Standby bool `json:"standby,omitempty"`

	// Authenticated reports whether the configured authentication was accepted.
	// +optional
	Authenticated bool `json:"authenticated,omitempty"`

	// Conditions represent the latest available observations of the connection.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=openbao
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec.address`
// +kubebuilder:printcolumn:name="Authenticated",type=boolean,JSONPath=`.status.authenticated`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenBaoConnection is the Schema for the openbaoconnections API.
type OpenBaoConnection struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OpenBaoConnection
	// +required
	Spec OpenBaoConnectionSpec `json:"spec"`

	// status defines the observed state of OpenBaoConnection
	// +optional
	Status OpenBaoConnectionStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenBaoConnectionList contains a list of OpenBaoConnection
type OpenBaoConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenBaoConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OpenBaoConnection{}, &OpenBaoConnectionList{})
		return nil
	})
}
