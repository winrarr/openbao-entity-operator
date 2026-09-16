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

// Package openbaoclient contains the small typed HTTP surface used by the
// operator. It deliberately models only the OpenBao endpoints that controllers
// currently reconcile.
package openbaoclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const maxErrorBodySize = 1 << 20

// Client is a typed client for the OpenBao HTTP API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
}

// HTTPError represents a non-successful OpenBao response.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("OpenBao API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("OpenBao API returned HTTP %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an OpenBao 404 response.
func IsNotFound(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound
}

// IsUnauthorized reports whether err represents an authentication or
// authorization failure.
func IsUnauthorized(err error) bool {
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden
}

// New validates the OpenBao address and returns an authenticated client.
func New(baseURL, token string, timeout time.Duration, caBundle []byte) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("OpenBao token is empty")
	}

	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return nil, fmt.Errorf("parse OpenBao address: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("OpenBao address must use http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("OpenBao address has no host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("OpenBao address must not contain a query or fragment")
	}
	if strings.HasSuffix(parsed.Path, "/v1") {
		return nil, errors.New("OpenBao address must not include the /v1 API prefix")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if len(caBundle) > 0 {
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(caBundle) {
			return nil, errors.New("OpenBao CA bundle contains no valid certificates")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}

	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		token: token,
	}, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, target any, allowedStatuses ...int) error {
	return c.doSegments(ctx, method, strings.Split(strings.Trim(path, "/"), "/"), body, target, allowedStatuses...)
}

func (c *Client) doSegments(ctx context.Context, method string, segments []string, body, target any, allowedStatuses ...int) error {
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + "/v1"
	requestURL.RawPath = strings.TrimRight(c.baseURL.EscapedPath(), "/") + "/v1"
	for _, segment := range segments {
		requestURL.Path += "/" + segment
		requestURL.RawPath += "/" + url.PathEscape(segment)
	}

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode OpenBao API request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
	if err != nil {
		return fmt.Errorf("create OpenBao API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("X-Vault-Request", "true")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call OpenBao API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if !isAllowedStatus(resp.StatusCode, allowedStatuses) {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		if readErr != nil {
			return fmt.Errorf("read OpenBao API error response: %w", readErr)
		}
		return &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(bodyBytes))}
	}
	if target == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode OpenBao API response: %w", err)
	}
	return nil
}

func isAllowedStatus(status int, allowed []int) bool {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return true
	}
	return slices.Contains(allowed, status)
}

type apiResponse[T any] struct {
	Data T `json:"data"`
}
