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

const auditPathSegment = "audit"

// Mount is the common OpenBao auth-method and secret-engine mount response.
type Mount struct {
	Accessor        string         `json:"accessor"`
	Config          map[string]any `json:"config"`
	Description     string         `json:"description"`
	ExternalEntropy bool           `json:"external_entropy_access"`
	Local           bool           `json:"local"`
	Options         map[string]any `json:"options"`
	PluginVersion   string         `json:"plugin_version"`
	SealWrap        bool           `json:"seal_wrap"`
	Type            string         `json:"type"`
	UUID            string         `json:"uuid"`
}

func mountPath(root, path string) []string {
	segments := append([]string{systemPathSegment, root}, strings.Split(path, "/")...)
	return segments
}

// GetAuthMethod reads an auth-method mount configuration.
func (c *Client) GetAuthMethod(ctx context.Context, path string) (*Mount, error) {
	var response apiResponse[Mount]
	if err := c.doSegments(ctx, http.MethodGet, mountPath("auth", path), nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// EnableAuthMethod enables an auth method at path.
func (c *Client) EnableAuthMethod(ctx context.Context, path string, request map[string]any) error {
	return c.doSegments(ctx, http.MethodPost, mountPath("auth", path), request, nil)
}

// TuneAuthMethod tunes an auth method at path.
func (c *Client) TuneAuthMethod(ctx context.Context, path string, request map[string]any) error {
	return c.doSegments(ctx, http.MethodPost, append(mountPath("auth", path), "tune"), request, nil)
}

// DisableAuthMethod disables an auth method at path.
func (c *Client) DisableAuthMethod(ctx context.Context, path string) error {
	return c.doSegments(ctx, http.MethodDelete, mountPath("auth", path), nil, nil)
}

// GetSecretEngine reads a secret-engine mount configuration.
func (c *Client) GetSecretEngine(ctx context.Context, path string) (*Mount, error) {
	var response apiResponse[Mount]
	if err := c.doSegments(ctx, http.MethodGet, mountPath("mounts", path), nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// EnableSecretEngine enables a secret engine at path.
func (c *Client) EnableSecretEngine(ctx context.Context, path string, request map[string]any) error {
	return c.doSegments(ctx, http.MethodPost, mountPath("mounts", path), request, nil)
}

// TuneSecretEngine tunes a secret engine at path.
func (c *Client) TuneSecretEngine(ctx context.Context, path string, request map[string]any) error {
	return c.doSegments(ctx, http.MethodPost, append(mountPath("mounts", path), "tune"), request, nil)
}

// DisableSecretEngine disables a secret engine at path.
func (c *Client) DisableSecretEngine(ctx context.Context, path string) error {
	return c.doSegments(ctx, http.MethodDelete, mountPath("mounts", path), nil, nil)
}

// Namespace is the observed OpenBao namespace configuration.
type Namespace struct {
	ID             string         `json:"id"`
	Path           string         `json:"path"`
	CustomMetadata map[string]any `json:"custom_metadata"`
	UUID           string         `json:"uuid"`
}

func namespacePath(path string) []string {
	return append([]string{systemPathSegment, "namespaces"}, strings.Split(path, "/")...)
}

func (c *Client) GetNamespace(ctx context.Context, path string) (*Namespace, error) {
	var response apiResponse[Namespace]
	if err := c.doSegments(ctx, http.MethodGet, namespacePath(path), nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}
func (c *Client) WriteNamespace(ctx context.Context, path string, request map[string]any) error {
	return c.doSegments(ctx, http.MethodPost, namespacePath(path), request, nil)
}
func (c *Client) DeleteNamespace(ctx context.Context, path string) error {
	return c.doSegments(ctx, http.MethodDelete, namespacePath(path), nil, nil)
}

// AuditDevice is an enabled audit backend returned by /sys/audit.
type AuditDevice struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Local       bool           `json:"local"`
	Options     map[string]any `json:"options"`
}

func (c *Client) GetAuditDevice(ctx context.Context, path string) (*AuditDevice, error) {
	var response apiResponse[map[string]AuditDevice]
	if err := c.doSegments(ctx, http.MethodGet, []string{systemPathSegment, auditPathSegment}, nil, &response); err != nil {
		return nil, err
	}
	device, ok := response.Data[path]
	if !ok {
		device, ok = response.Data[strings.TrimSuffix(path, "/")+"/"]
	}
	if !ok {
		return nil, &HTTPError{StatusCode: http.StatusNotFound, Body: "audit device not found"}
	}
	return &device, nil
}
func (c *Client) WriteAuditDevice(ctx context.Context, path string, request map[string]any) error {
	return c.doSegments(ctx, http.MethodPost, append([]string{systemPathSegment, auditPathSegment}, path), request, nil)
}
func (c *Client) DeleteAuditDevice(ctx context.Context, path string) error {
	return c.doSegments(ctx, http.MethodDelete, append([]string{systemPathSegment, auditPathSegment}, path), nil, nil)
}

// RateLimitQuota is the stored rate-limit quota configuration.
type RateLimitQuota map[string]any

func quotaPath(name string) []string {
	return []string{systemPathSegment, "quotas", "rate-limit", name}
}
func (c *Client) GetRateLimitQuota(ctx context.Context, name string) (RateLimitQuota, error) {
	var response apiResponse[RateLimitQuota]
	if err := c.doSegments(ctx, http.MethodGet, quotaPath(name), nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}
func (c *Client) WriteRateLimitQuota(ctx context.Context, name string, request RateLimitQuota) error {
	return c.doSegments(ctx, http.MethodPost, quotaPath(name), request, nil)
}
func (c *Client) DeleteRateLimitQuota(ctx context.Context, name string) error {
	return c.doSegments(ctx, http.MethodDelete, quotaPath(name), nil, nil)
}

// Workflow is the stored workflow response.
type Workflow map[string]any

func workflowPath(path string) []string {
	return append([]string{systemPathSegment, "workflows", "manage"}, strings.Split(path, "/")...)
}
func (c *Client) GetWorkflow(ctx context.Context, path string) (Workflow, error) {
	var response apiResponse[Workflow]
	if err := c.doSegments(ctx, http.MethodGet, workflowPath(path), nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}
func (c *Client) WriteWorkflow(ctx context.Context, path string, request Workflow) error {
	return c.doSegments(ctx, http.MethodPost, workflowPath(path), request, nil)
}
func (c *Client) DeleteWorkflow(ctx context.Context, path string) error {
	return c.doSegments(ctx, http.MethodDelete, workflowPath(path), nil, nil)
}

// Plugin is the registered plugin response.
type Plugin map[string]any

func pluginPath(typ, name string) []string {
	return []string{systemPathSegment, "plugins", "catalog", typ, name}
}
func (c *Client) GetPlugin(ctx context.Context, typ, name string) (Plugin, error) {
	var response apiResponse[Plugin]
	if err := c.doSegments(ctx, http.MethodGet, pluginPath(typ, name), nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}
func (c *Client) WritePlugin(ctx context.Context, typ, name string, request Plugin) error {
	return c.doSegments(ctx, http.MethodPost, pluginPath(typ, name), request, nil)
}
func (c *Client) DeletePlugin(ctx context.Context, typ, name string) error {
	return c.doSegments(ctx, http.MethodDelete, pluginPath(typ, name), nil, nil)
}
