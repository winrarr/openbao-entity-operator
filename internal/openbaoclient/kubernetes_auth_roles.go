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

package openbaoclient

import (
	"context"
	"net/http"
	"strings"
)

// KubernetesAuthRole is the supported OpenBao Kubernetes Auth role surface.
// The token-prefixed fields are the stable wire names used by OpenBao's role
// endpoint; Policies is retained for responses from older plugin versions.
type KubernetesAuthRole struct {
	BoundServiceAccountNames      []string `json:"bound_service_account_names"`
	BoundServiceAccountNamespaces []string `json:"bound_service_account_namespaces"`
	TokenPolicies                 []string `json:"token_policies"`
	Policies                      []string `json:"policies"`
	TokenTTL                      int64    `json:"token_ttl"`
	TokenMaxTTL                   int64    `json:"token_max_ttl"`
	TokenPeriod                   int64    `json:"token_period"`
	Audience                      string   `json:"audience"`
	TokenType                     string   `json:"token_type"`
	TokenNumUses                  int64    `json:"token_num_uses"`
	TokenNoDefaultPolicy          bool     `json:"token_no_default_policy"`
	TokenExplicitMaxTTL           int64    `json:"token_explicit_max_ttl"`
	TokenBoundCIDRs               []string `json:"token_bound_cidrs"`
}

// EffectivePolicies returns the policy list returned by the role endpoint.
func (r *KubernetesAuthRole) EffectivePolicies() []string {
	if len(r.TokenPolicies) > 0 {
		return r.TokenPolicies
	}
	return r.Policies
}

// KubernetesAuthRoleRequest is the mutable OpenBao Kubernetes Auth role configuration.
type KubernetesAuthRoleRequest struct {
	BoundServiceAccountNames      []string `json:"bound_service_account_names"`
	BoundServiceAccountNamespaces []string `json:"bound_service_account_namespaces"`
	TokenPolicies                 []string `json:"token_policies"`
	TokenTTL                      int64    `json:"token_ttl"`
	TokenMaxTTL                   int64    `json:"token_max_ttl"`
	TokenPeriod                   int64    `json:"token_period"`
	Audience                      string   `json:"audience,omitempty"`
	TokenType                     string   `json:"token_type,omitempty"`
	TokenNumUses                  *int64   `json:"token_num_uses,omitempty"`
	TokenNoDefaultPolicy          *bool    `json:"token_no_default_policy,omitempty"`
	TokenExplicitMaxTTL           *int64   `json:"token_explicit_max_ttl,omitempty"`
	TokenBoundCIDRs               []string `json:"token_bound_cidrs,omitempty"`
}

func kubernetesAuthRolePath(mountPath, name string) []string {
	segments := append([]string{authPathSegment}, strings.Split(mountPath, "/")...)
	return append(segments, "role", name)
}

// GetKubernetesAuthRole reads a role from an OpenBao Kubernetes Auth mount.
func (c *Client) GetKubernetesAuthRole(ctx context.Context, mountPath, name string) (*KubernetesAuthRole, error) {
	if err := validateAuthMountPath(mountPath, "Kubernetes auth"); err != nil {
		return nil, err
	}
	var response apiResponse[KubernetesAuthRole]
	if err := c.doSegments(ctx, http.MethodGet, kubernetesAuthRolePath(mountPath, name), nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// WriteKubernetesAuthRole creates or replaces a role in an OpenBao Kubernetes Auth mount.
func (c *Client) WriteKubernetesAuthRole(ctx context.Context, mountPath, name string, request KubernetesAuthRoleRequest) error {
	if err := validateAuthMountPath(mountPath, "Kubernetes auth"); err != nil {
		return err
	}
	return c.doSegments(ctx, http.MethodPost, kubernetesAuthRolePath(mountPath, name), request, nil)
}

// DeleteKubernetesAuthRole deletes a role from an OpenBao Kubernetes Auth mount.
func (c *Client) DeleteKubernetesAuthRole(ctx context.Context, mountPath, name string) error {
	if err := validateAuthMountPath(mountPath, "Kubernetes auth"); err != nil {
		return err
	}
	return c.doSegments(ctx, http.MethodDelete, kubernetesAuthRolePath(mountPath, name), nil, nil)
}
