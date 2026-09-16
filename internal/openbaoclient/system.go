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
