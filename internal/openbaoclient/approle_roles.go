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

// AppRole is the supported OpenBao AppRole configuration. It intentionally
// excludes role-id and Secret ID issuance; those are credential-delivery APIs.
type AppRole struct {
	BindSecretID         *bool    `json:"bind_secret_id"`
	LocalSecretIDs       *bool    `json:"local_secret_ids"`
	SecretIDBoundCIDRs   []string `json:"secret_id_bound_cidrs"`
	SecretIDNumUses      *int64   `json:"secret_id_num_uses"`
	SecretIDTTL          *int64   `json:"secret_id_ttl"`
	TokenBoundCIDRs      []string `json:"token_bound_cidrs"`
	TokenExplicitMaxTTL  *int64   `json:"token_explicit_max_ttl"`
	TokenMaxTTL          *int64   `json:"token_max_ttl"`
	TokenNoDefaultPolicy *bool    `json:"token_no_default_policy"`
	TokenNumUses         *int64   `json:"token_num_uses"`
	TokenPeriod          *int64   `json:"token_period"`
	TokenPolicies        []string `json:"token_policies"`
	Policies             []string `json:"policies"`
	TokenTTL             *int64   `json:"token_ttl"`
	TokenType            string   `json:"token_type"`
}

// AppRoleRequest is the mutable OpenBao AppRole configuration.
type AppRoleRequest struct {
	BindSecretID         *bool    `json:"bind_secret_id,omitempty"`
	LocalSecretIDs       *bool    `json:"local_secret_ids,omitempty"`
	SecretIDBoundCIDRs   []string `json:"secret_id_bound_cidrs,omitempty"`
	SecretIDNumUses      *int64   `json:"secret_id_num_uses,omitempty"`
	SecretIDTTL          *int64   `json:"secret_id_ttl,omitempty"`
	TokenBoundCIDRs      []string `json:"token_bound_cidrs,omitempty"`
	TokenExplicitMaxTTL  *int64   `json:"token_explicit_max_ttl,omitempty"`
	TokenMaxTTL          *int64   `json:"token_max_ttl,omitempty"`
	TokenNoDefaultPolicy *bool    `json:"token_no_default_policy,omitempty"`
	TokenNumUses         *int64   `json:"token_num_uses,omitempty"`
	TokenPeriod          *int64   `json:"token_period,omitempty"`
	TokenPolicies        []string `json:"token_policies,omitempty"`
	TokenTTL             *int64   `json:"token_ttl,omitempty"`
	TokenType            string   `json:"token_type,omitempty"`
}

// EffectivePolicies returns the policy list returned by the role endpoint.
func (r *AppRole) EffectivePolicies() []string {
	if len(r.TokenPolicies) > 0 {
		return r.TokenPolicies
	}
	return r.Policies
}

func appRolePath(mountPath, name string) []string {
	segments := append([]string{authPathSegment}, strings.Split(mountPath, "/")...)
	return append(segments, "role", name)
}

// GetAppRole reads an AppRole configuration.
func (c *Client) GetAppRole(ctx context.Context, mountPath, name string) (*AppRole, error) {
	if err := validateAuthMountPath(mountPath, "AppRole auth"); err != nil {
		return nil, err
	}
	var response apiResponse[AppRole]
	if err := c.doSegments(ctx, http.MethodGet, appRolePath(mountPath, name), nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// WriteAppRole creates or replaces an AppRole configuration.
func (c *Client) WriteAppRole(ctx context.Context, mountPath, name string, request AppRoleRequest) error {
	if err := validateAuthMountPath(mountPath, "AppRole auth"); err != nil {
		return err
	}
	return c.doSegments(ctx, http.MethodPost, appRolePath(mountPath, name), request, nil)
}

// DeleteAppRole deletes an AppRole configuration.
func (c *Client) DeleteAppRole(ctx context.Context, mountPath, name string) error {
	if err := validateAuthMountPath(mountPath, "AppRole auth"); err != nil {
		return err
	}
	return c.doSegments(ctx, http.MethodDelete, appRolePath(mountPath, name), nil, nil)
}
