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
	"fmt"
	"net/http"
)

const (
	oidcPathSegment       = "oidc"
	oidcConfigSegment     = "config"
	oidcProviderSegment   = "provider"
	oidcClientSegment     = "client"
	oidcKeySegment        = "key"
	oidcRoleSegment       = "role"
	oidcScopeSegment      = "scope"
	oidcAssignmentSegment = "assignment"
)

// OIDCObject is the dynamic configuration returned by OpenBao's OIDC endpoints.
// The endpoint schemas are intentionally extensible across OpenBao releases;
// the public Kubernetes APIs still expose typed fields for the supported surface.
type OIDCObject map[string]any

func oidcResourceSegment(resource string) (string, error) {
	switch resource {
	case oidcProviderSegment, oidcClientSegment, oidcKeySegment, oidcRoleSegment, oidcScopeSegment, oidcAssignmentSegment:
		return resource, nil
	default:
		return "", fmt.Errorf("unsupported OpenBao OIDC resource %q", resource)
	}
}

func oidcResourcePath(resource, name string) ([]string, error) {
	segment, err := oidcResourceSegment(resource)
	if err != nil {
		return nil, err
	}
	return []string{identityPathSegment, oidcPathSegment, segment, name}, nil
}

func oidcConfigPath() []string {
	return []string{identityPathSegment, oidcPathSegment, oidcConfigSegment}
}

// GetOIDCResource reads a named OIDC configuration object.
func (c *Client) GetOIDCResource(ctx context.Context, resource, name string) (OIDCObject, error) {
	path, err := oidcResourcePath(resource, name)
	if err != nil {
		return nil, err
	}
	var response apiResponse[OIDCObject]
	if err := c.doSegments(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

// WriteOIDCResource creates or replaces a named OIDC configuration object.
func (c *Client) WriteOIDCResource(ctx context.Context, resource, name string, request OIDCObject) error {
	path, err := oidcResourcePath(resource, name)
	if err != nil {
		return err
	}
	return c.doSegments(ctx, http.MethodPost, path, request, nil)
}

// DeleteOIDCResource deletes a named OIDC configuration object.
func (c *Client) DeleteOIDCResource(ctx context.Context, resource, name string) error {
	path, err := oidcResourcePath(resource, name)
	if err != nil {
		return err
	}
	return c.doSegments(ctx, http.MethodDelete, path, nil, nil)
}

// GetOIDCConfig reads the singleton OIDC configuration.
func (c *Client) GetOIDCConfig(ctx context.Context) (OIDCObject, error) {
	var response apiResponse[OIDCObject]
	if err := c.doSegments(ctx, http.MethodGet, oidcConfigPath(), nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

// WriteOIDCConfig updates the singleton OIDC configuration.
func (c *Client) WriteOIDCConfig(ctx context.Context, request OIDCObject) error {
	return c.doSegments(ctx, http.MethodPost, oidcConfigPath(), request, nil)
}
