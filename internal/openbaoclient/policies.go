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

const (
	systemPathSegment   = "sys"
	policiesPathSegment = "policies"
	aclPathSegment      = "acl"
)

// Policy is the OpenBao ACL policy surface reconciled by this project.
type Policy struct {
	Name    string `json:"name"`
	Rules   string `json:"policy"`
	Version int64  `json:"version"`
}

// PolicyRequest is the mutable OpenBao ACL policy configuration.
type PolicyRequest struct {
	Rules string `json:"policy"`
}

func policyPath(name string) []string {
	return []string{systemPathSegment, policiesPathSegment, aclPathSegment, name}
}

// GetPolicy reads an ACL policy by its OpenBao name.
func (c *Client) GetPolicy(ctx context.Context, name string) (*Policy, error) {
	var response apiResponse[Policy]
	if err := c.doSegments(ctx, http.MethodGet, policyPath(name), nil, &response); err != nil {
		return nil, err
	}
	response.Data.Name = firstNonEmpty(response.Data.Name, name)
	return &response.Data, nil
}

// WritePolicy creates or replaces an ACL policy by name.
func (c *Client) WritePolicy(ctx context.Context, name string, request PolicyRequest) error {
	return c.doSegments(ctx, http.MethodPost, policyPath(name), request, nil)
}

// DeletePolicy deletes an ACL policy by name.
func (c *Client) DeletePolicy(ctx context.Context, name string) error {
	return c.doSegments(ctx, http.MethodDelete, policyPath(name), nil, nil)
}
