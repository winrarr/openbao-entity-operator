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

// PasswordPolicy is the OpenBao password-generation policy surface.
type PasswordPolicy struct {
	Rules string
}

func passwordPolicyPath(name string) []string {
	return []string{systemPathSegment, policiesPathSegment, "password", name}
}

// GetPasswordPolicy reads a password policy by name. OpenBao releases have
// returned both envelope and top-level forms for this endpoint, so the client
// accepts either response shape.
func (c *Client) GetPasswordPolicy(ctx context.Context, name string) (*PasswordPolicy, error) {
	var response struct {
		Policy string `json:"policy"`
		Data   struct {
			Policy string `json:"policy"`
		} `json:"data"`
	}
	if err := c.doSegments(ctx, http.MethodGet, passwordPolicyPath(name), nil, &response); err != nil {
		return nil, err
	}
	if response.Policy == "" {
		response.Policy = response.Data.Policy
	}
	return &PasswordPolicy{Rules: response.Policy}, nil
}

// WritePasswordPolicy creates or replaces a password policy.
func (c *Client) WritePasswordPolicy(ctx context.Context, name, rules string) error {
	return c.doSegments(ctx, http.MethodPost, passwordPolicyPath(name), map[string]string{"policy": rules}, nil)
}

// DeletePasswordPolicy deletes a password policy by name.
func (c *Client) DeletePasswordPolicy(ctx context.Context, name string) error {
	return c.doSegments(ctx, http.MethodDelete, passwordPolicyPath(name), nil, nil)
}
