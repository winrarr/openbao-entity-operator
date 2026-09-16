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
	"strings"
	"time"
)

// Health is the subset of /sys/health needed by the connection status.
type Health struct {
	Version     string `json:"version"`
	Initialized bool   `json:"initialized"`
	Sealed      bool   `json:"sealed"`
	Standby     bool   `json:"standby"`
}

// CheckHealth reads OpenBao health. Sealed, uninitialized, standby, and DR
// states are returned by OpenBao with non-2xx statuses and are still decoded.
func (c *Client) CheckHealth(ctx context.Context) (*Health, error) {
	var response Health
	if err := c.do(ctx, http.MethodGet, "sys/health", nil, &response,
		http.StatusTooManyRequests,
		472,
		473,
		http.StatusNotImplemented,
		http.StatusServiceUnavailable,
	); err != nil {
		return nil, err
	}
	return &response, nil
}

// LookupSelf validates the configured token against OpenBao's token auth API.
func (c *Client) LookupSelf(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "auth/token/lookup-self", nil, nil)
}

type tokenAuth struct {
	ClientToken   string `json:"client_token"`
	LeaseDuration int64  `json:"lease_duration"`
	Renewable     bool   `json:"renewable"`
}

type tokenAuthResponse struct {
	Auth tokenAuth `json:"auth"`
}

func (c *Client) loginKubernetes(ctx context.Context, jwt string) (string, tokenLease, error) {
	segments := append([]string{authPathSegment}, strings.Split(c.kubernetesAuth.MountPath, "/")...)
	segments = append(segments, "login")
	var response tokenAuthResponse
	if err := c.doSegmentsQueryWithToken(ctx, http.MethodPost, segments, nil, map[string]string{
		"jwt":  jwt,
		"role": c.kubernetesAuth.Role,
	}, &response, ""); err != nil {
		return "", tokenLease{}, err
	}
	if strings.TrimSpace(response.Auth.ClientToken) == "" {
		return "", tokenLease{}, fmt.Errorf("OpenBao Kubernetes auth response did not include a client token")
	}
	return response.Auth.ClientToken, newTokenLease(response.Auth, c.now()), nil
}

func (c *Client) loginAppRole(ctx context.Context, roleID, secretID string) (string, tokenLease, error) {
	segments := append([]string{authPathSegment}, strings.Split(c.appRoleAuth.MountPath, "/")...)
	segments = append(segments, "login")
	var response tokenAuthResponse
	if err := c.doSegmentsQueryWithToken(ctx, http.MethodPost, segments, nil, map[string]string{
		"role_id":   roleID,
		"secret_id": secretID,
	}, &response, ""); err != nil {
		return "", tokenLease{}, err
	}
	if strings.TrimSpace(response.Auth.ClientToken) == "" {
		return "", tokenLease{}, fmt.Errorf("OpenBao AppRole auth response did not include a client token")
	}
	return response.Auth.ClientToken, newTokenLease(response.Auth, c.now()), nil
}

func (c *Client) renewToken(ctx context.Context, token string) (tokenLease, error) {
	var response tokenAuthResponse
	if err := c.doSegmentsQueryWithToken(ctx, http.MethodPost, []string{authPathSegment, "token", "renew-self"}, nil, map[string]string{
		"increment": "",
	}, &response, token); err != nil {
		return tokenLease{}, err
	}
	return newTokenLease(response.Auth, c.now()), nil
}

func newTokenLease(auth tokenAuth, now time.Time) tokenLease {
	lease := tokenLease{renewable: auth.Renewable}
	if auth.LeaseDuration > 0 {
		lease.duration = time.Duration(auth.LeaseDuration) * time.Second
		lease.expiresAt = now.Add(lease.duration)
	}
	return lease
}
