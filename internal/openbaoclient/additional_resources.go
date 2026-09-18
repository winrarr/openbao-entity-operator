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

// AdditionalObject is the response shape used by OpenBao's small durable
// configuration endpoints whose fields vary by provider or deployment.
type AdditionalObject map[string]any

func (c *Client) getAdditional(ctx context.Context, segments ...string) (AdditionalObject, error) {
	var response apiResponse[AdditionalObject]
	if err := c.doSegments(ctx, http.MethodGet, segments, nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *Client) writeAdditional(ctx context.Context, segments []string, request AdditionalObject) error {
	return c.doSegments(ctx, http.MethodPost, segments, request, nil)
}

func (c *Client) deleteAdditional(ctx context.Context, segments ...string) error {
	return c.doSegments(ctx, http.MethodDelete, segments, nil, nil)
}

// GetMFALoginEnforcement reads one MFA login enforcement.
func (c *Client) GetMFALoginEnforcement(ctx context.Context, name string) (AdditionalObject, error) {
	return c.getAdditional(ctx, identityPathSegment, "mfa", "login-enforcement", name)
}
func (c *Client) WriteMFALoginEnforcement(ctx context.Context, name string, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{identityPathSegment, "mfa", "login-enforcement", name}, request)
}
func (c *Client) DeleteMFALoginEnforcement(ctx context.Context, name string) error {
	return c.deleteAdditional(ctx, identityPathSegment, "mfa", "login-enforcement", name)
}

// GetMFAMethod reads one typed MFA method.
func (c *Client) GetMFAMethod(ctx context.Context, methodType, methodID string) (AdditionalObject, error) {
	return c.getAdditional(ctx, identityPathSegment, "mfa", "method", methodType, methodID)
}
func (c *Client) WriteMFAMethod(ctx context.Context, methodType, methodID string, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{identityPathSegment, "mfa", "method", methodType, methodID}, request)
}
func (c *Client) DeleteMFAMethod(ctx context.Context, methodType, methodID string) error {
	return c.deleteAdditional(ctx, identityPathSegment, "mfa", "method", methodType, methodID)
}

func (c *Client) GetCORSConfiguration(ctx context.Context) (AdditionalObject, error) {
	return c.getAdditional(ctx, systemPathSegment, oidcConfigSegment, "cors")
}
func (c *Client) WriteCORSConfiguration(ctx context.Context, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{systemPathSegment, oidcConfigSegment, "cors"}, request)
}
func (c *Client) DeleteCORSConfiguration(ctx context.Context) error {
	return c.deleteAdditional(ctx, systemPathSegment, oidcConfigSegment, "cors")
}

func (c *Client) GetAuditRequestHeader(ctx context.Context, header string) (AdditionalObject, error) {
	return c.getAdditional(ctx, systemPathSegment, oidcConfigSegment, "auditing", "request-headers", header)
}
func (c *Client) WriteAuditRequestHeader(ctx context.Context, header string, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{systemPathSegment, oidcConfigSegment, "auditing", "request-headers", header}, request)
}
func (c *Client) DeleteAuditRequestHeader(ctx context.Context, header string) error {
	return c.deleteAdditional(ctx, systemPathSegment, oidcConfigSegment, "auditing", "request-headers", header)
}

func (c *Client) GetUIHeader(ctx context.Context, header string) (AdditionalObject, error) {
	return c.getAdditional(ctx, systemPathSegment, oidcConfigSegment, "ui", "headers", header)
}
func (c *Client) WriteUIHeader(ctx context.Context, header string, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{systemPathSegment, oidcConfigSegment, "ui", "headers", header}, request)
}
func (c *Client) DeleteUIHeader(ctx context.Context, header string) error {
	return c.deleteAdditional(ctx, systemPathSegment, oidcConfigSegment, "ui", "headers", header)
}

func (c *Client) GetRateLimitQuotaConfiguration(ctx context.Context) (AdditionalObject, error) {
	return c.getAdditional(ctx, systemPathSegment, "quotas", oidcConfigSegment)
}
func (c *Client) WriteRateLimitQuotaConfiguration(ctx context.Context, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{systemPathSegment, "quotas", oidcConfigSegment}, request)
}

func (c *Client) GetLogger(ctx context.Context, name string) (AdditionalObject, error) {
	if name == "" {
		return c.getAdditional(ctx, systemPathSegment, "loggers")
	}
	value, err := c.getAdditional(ctx, systemPathSegment, "loggers", name)
	if err != nil {
		return nil, err
	}
	if level, ok := value[name].(string); ok {
		return AdditionalObject{"level": level}, nil
	}
	return value, nil
}
func (c *Client) WriteLogger(ctx context.Context, name string, request AdditionalObject) error {
	if name == "" {
		return c.writeAdditional(ctx, []string{systemPathSegment, "loggers"}, request)
	}
	return c.writeAdditional(ctx, []string{systemPathSegment, "loggers", name}, request)
}
func (c *Client) DeleteLogger(ctx context.Context, name string) error {
	if name == "" {
		return c.deleteAdditional(ctx, systemPathSegment, "loggers")
	}
	return c.deleteAdditional(ctx, systemPathSegment, "loggers", name)
}

func (c *Client) GetEncryptionKeyConfiguration(ctx context.Context) (AdditionalObject, error) {
	return c.getAdditional(ctx, systemPathSegment, "rotate", oidcConfigSegment)
}
func (c *Client) WriteEncryptionKeyConfiguration(ctx context.Context, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{systemPathSegment, "rotate", oidcConfigSegment}, request)
}

func (c *Client) GetKeyringRotationConfiguration(ctx context.Context) (AdditionalObject, error) {
	return c.getAdditional(ctx, systemPathSegment, "rotate", "keyring", oidcConfigSegment)
}
func (c *Client) WriteKeyringRotationConfiguration(ctx context.Context, request AdditionalObject) error {
	return c.writeAdditional(ctx, []string{systemPathSegment, "rotate", "keyring", oidcConfigSegment}, request)
}
