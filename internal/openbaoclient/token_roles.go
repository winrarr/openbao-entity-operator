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
)

// TokenRole is the supported OpenBao token role configuration.
type TokenRole struct {
	AllowedEntityAliases   []string `json:"allowed_entity_aliases"`
	AllowedPolicies        []string `json:"allowed_policies"`
	AllowedPoliciesGlob    []string `json:"allowed_policies_glob"`
	DisallowedPolicies     []string `json:"disallowed_policies"`
	DisallowedPoliciesGlob []string `json:"disallowed_policies_glob"`
	TokenBoundCIDRs        []string `json:"token_bound_cidrs"`
	TokenExplicitMaxTTL    *int64   `json:"token_explicit_max_ttl"`
	TokenPeriod            *int64   `json:"token_period"`
	TokenNumUses           *int64   `json:"token_num_uses"`
	TokenNoDefaultPolicy   *bool    `json:"token_no_default_policy"`
	TokenType              string   `json:"token_type"`
	Orphan                 *bool    `json:"orphan"`
	Renewable              *bool    `json:"renewable"`
	PathSuffix             string   `json:"path_suffix"`
}

// TokenRoleRequest is the mutable OpenBao token role configuration.
type TokenRoleRequest struct {
	AllowedEntityAliases   []string `json:"allowed_entity_aliases,omitempty"`
	AllowedPolicies        []string `json:"allowed_policies,omitempty"`
	AllowedPoliciesGlob    []string `json:"allowed_policies_glob,omitempty"`
	DisallowedPolicies     []string `json:"disallowed_policies,omitempty"`
	DisallowedPoliciesGlob []string `json:"disallowed_policies_glob,omitempty"`
	TokenBoundCIDRs        []string `json:"token_bound_cidrs,omitempty"`
	TokenExplicitMaxTTL    *int64   `json:"token_explicit_max_ttl,omitempty"`
	TokenPeriod            *int64   `json:"token_period,omitempty"`
	TokenNumUses           *int64   `json:"token_num_uses,omitempty"`
	TokenNoDefaultPolicy   *bool    `json:"token_no_default_policy,omitempty"`
	TokenType              string   `json:"token_type,omitempty"`
	Orphan                 *bool    `json:"orphan,omitempty"`
	Renewable              *bool    `json:"renewable,omitempty"`
	PathSuffix             string   `json:"path_suffix,omitempty"`
}

func tokenRolePath(name string) []string {
	return []string{authPathSegment, "token", "roles", name}
}

// GetTokenRole reads a token role by name.
func (c *Client) GetTokenRole(ctx context.Context, name string) (*TokenRole, error) {
	var response apiResponse[TokenRole]
	if err := c.doSegments(ctx, http.MethodGet, tokenRolePath(name), nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// WriteTokenRole creates or replaces a token role.
func (c *Client) WriteTokenRole(ctx context.Context, name string, request TokenRoleRequest) error {
	return c.doSegments(ctx, http.MethodPost, tokenRolePath(name), request, nil)
}

// DeleteTokenRole deletes a token role.
func (c *Client) DeleteTokenRole(ctx context.Context, name string) error {
	return c.doSegments(ctx, http.MethodDelete, tokenRolePath(name), nil, nil)
}
