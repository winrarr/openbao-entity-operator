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
	"os"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	maxErrorBodySize        = 1 << 20
	authPathSegment         = "auth"
	defaultKubernetesMount  = "kubernetes"
	defaultAppRoleMount     = "approle"
	serviceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// TokenSource returns a short-lived Kubernetes ServiceAccount JWT.
type TokenSource func(context.Context) (string, error)

// CredentialSource returns a credential when a fresh AppRole login is needed.
// Sources are deliberately lazy so rotated Kubernetes Secrets are picked up
// without rebuilding or restarting the operator.
type CredentialSource func(context.Context) (string, error)

// KubernetesAuthOptions configures the OpenBao Kubernetes auth login.
type KubernetesAuthOptions struct {
	// MountPath is the auth mount path without the leading auth/ prefix.
	MountPath string
	Role      string
	JWTSource TokenSource
}

// AppRoleAuthOptions configures OpenBao AppRole login.
type AppRoleAuthOptions struct {
	// MountPath is the auth mount path without the leading auth/ prefix.
	MountPath string
	RoleID    CredentialSource
	SecretID  CredentialSource
}

type tokenLease struct {
	expiresAt time.Time
	duration  time.Duration
	renewable bool
}

// Client is a typed client for the OpenBao HTTP API.
type Client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	namespace      string
	kubernetesAuth *KubernetesAuthOptions
	appRoleAuth    *AppRoleAuthOptions
	tokenMu        sync.Mutex
	token          string
	tokenLease     tokenLease
	now            func() time.Time
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
	return NewWithNamespace(baseURL, token, timeout, caBundle, "")
}

// NewWithNamespace validates the OpenBao address and returns an authenticated
// client that routes requests through the supplied namespace.
func NewWithNamespace(baseURL, token string, timeout time.Duration, caBundle []byte, namespace string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("OpenBao token is empty")
	}
	client, err := newClient(baseURL, timeout, caBundle, namespace)
	if err != nil {
		return nil, err
	}
	client.token = token
	return client, nil
}

// NewWithKubernetesAuth returns a client that obtains and renews an OpenBao
// token using the Kubernetes auth method. The JWT source is called again when
// a re-login is needed, allowing projected ServiceAccount tokens to rotate.
func NewWithKubernetesAuth(baseURL string, options KubernetesAuthOptions, timeout time.Duration, caBundle []byte, namespace string) (*Client, error) {
	if strings.TrimSpace(options.Role) == "" {
		return nil, errors.New("OpenBao Kubernetes auth role is empty")
	}
	if strings.IndexFunc(options.Role, unicode.IsSpace) >= 0 {
		return nil, errors.New("OpenBao Kubernetes auth role must not contain whitespace")
	}
	if options.JWTSource == nil {
		return nil, errors.New("OpenBao Kubernetes auth JWT source is nil")
	}
	mountPath := options.MountPath
	if mountPath == "" {
		mountPath = defaultKubernetesMount
	}
	if err := validateAuthMountPath(mountPath, "Kubernetes auth"); err != nil {
		return nil, err
	}
	options.MountPath = mountPath
	client, err := newClient(baseURL, timeout, caBundle, namespace)
	if err != nil {
		return nil, err
	}
	client.kubernetesAuth = &options
	return client, nil
}

// NewWithAppRole returns a client that obtains and renews an OpenBao token
// using AppRole. The credential sources are called again when a re-login is
// needed, allowing Kubernetes Secrets to rotate without an operator restart.
func NewWithAppRole(baseURL string, options AppRoleAuthOptions, timeout time.Duration, caBundle []byte, namespace string) (*Client, error) {
	if options.RoleID == nil {
		return nil, errors.New("OpenBao AppRole role ID source is nil")
	}
	if options.SecretID == nil {
		return nil, errors.New("OpenBao AppRole Secret ID source is nil")
	}
	mountPath := options.MountPath
	if mountPath == "" {
		mountPath = defaultAppRoleMount
	}
	if err := validateAuthMountPath(mountPath, "AppRole auth"); err != nil {
		return nil, err
	}
	options.MountPath = mountPath
	client, err := newClient(baseURL, timeout, caBundle, namespace)
	if err != nil {
		return nil, err
	}
	client.appRoleAuth = &options
	return client, nil
}

func newClient(baseURL string, timeout time.Duration, caBundle []byte, namespace string) (*Client, error) {
	if err := validateNamespace(namespace); err != nil {
		return nil, err
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
		namespace: namespace,
		now:       time.Now,
	}, nil
}

// ServiceAccountTokenSource reads the projected ServiceAccount JWT used by
// the in-cluster Kubernetes auth flow.
func ServiceAccountTokenSource(context.Context) (string, error) {
	return readServiceAccountToken()
}

func readServiceAccountToken() (string, error) {
	data, err := os.ReadFile(serviceAccountTokenPath)
	if err != nil {
		return "", fmt.Errorf("read projected ServiceAccount token: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", errors.New("projected ServiceAccount token is empty")
	}
	return token, nil
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}
	if strings.TrimSpace(namespace) != namespace {
		return errors.New("OpenBao namespace must not start or end with whitespace")
	}
	reserved := map[string]struct{}{
		".": {}, "..": {}, "root": {}, "sys": {}, "audit": {},
		"auth": {}, "cubbyhole": {}, "identity": {},
	}
	for segment := range strings.SplitSeq(namespace, "/") {
		if segment == "" {
			return errors.New("OpenBao namespace must not contain empty path segments")
		}
		if _, ok := reserved[segment]; ok {
			return fmt.Errorf("OpenBao namespace segment %q is reserved", segment)
		}
		if strings.IndexFunc(segment, unicode.IsSpace) >= 0 {
			return errors.New("OpenBao namespace must not contain whitespace")
		}
	}
	return nil
}

func validateAuthMountPath(mountPath, authName string) error {
	if strings.TrimSpace(mountPath) != mountPath || mountPath == "" {
		return fmt.Errorf("OpenBao %s mount path must be non-empty and contain no surrounding whitespace", authName)
	}
	if strings.HasPrefix(mountPath, "auth/") {
		return fmt.Errorf("OpenBao %s mount path must omit the auth/ prefix", authName)
	}
	for segment := range strings.SplitSeq(mountPath, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.IndexFunc(segment, unicode.IsSpace) >= 0 {
			return fmt.Errorf("invalid OpenBao %s mount path %q", authName, mountPath)
		}
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body, target any, allowedStatuses ...int) error {
	return c.doSegments(ctx, method, strings.Split(strings.Trim(path, "/"), "/"), body, target, allowedStatuses...)
}

func (c *Client) doSegments(ctx context.Context, method string, segments []string, body, target any, allowedStatuses ...int) error {
	return c.doSegmentsQuery(ctx, method, segments, nil, body, target, allowedStatuses...)
}

func (c *Client) doSegmentsQuery(ctx context.Context, method string, segments []string, query url.Values, body, target any, allowedStatuses ...int) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}
	if err := c.doSegmentsQueryWithToken(ctx, method, segments, query, body, target, c.currentToken(), allowedStatuses...); err != nil {
		if !IsUnauthorized(err) || !c.usesDynamicAuth() {
			return err
		}
		c.invalidateToken()
		if err := c.ensureToken(ctx); err != nil {
			return err
		}
		return c.doSegmentsQueryWithToken(ctx, method, segments, query, body, target, c.currentToken(), allowedStatuses...)
	}
	return nil
}

func (c *Client) doSegmentsQueryWithToken(ctx context.Context, method string, segments []string, query url.Values, body, target any, token string, allowedStatuses ...int) error {
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + "/v1"
	requestURL.RawPath = strings.TrimRight(c.baseURL.EscapedPath(), "/") + "/v1"
	for _, segment := range segments {
		requestURL.Path += "/" + segment
		requestURL.RawPath += "/" + url.PathEscape(segment)
	}
	requestURL.RawQuery = query.Encode()

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
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("X-Vault-Request", "true")
	if c.namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.namespace)
	}
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

func (c *Client) currentToken() string {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	return c.token
}

func (c *Client) invalidateToken() {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.token = ""
	c.tokenLease = tokenLease{}
}

func (c *Client) ensureToken(ctx context.Context) error {
	if !c.usesDynamicAuth() {
		return nil
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.token != "" && !tokenNeedsRefresh(c.tokenLease, c.now()) {
		return nil
	}

	if c.token != "" && c.tokenLease.renewable {
		if lease, err := c.renewToken(ctx, c.token); err == nil {
			c.tokenLease = lease
			return nil
		}
	}

	var clientToken string
	var lease tokenLease
	var err error
	switch {
	case c.kubernetesAuth != nil:
		jwt, sourceErr := c.kubernetesAuth.JWTSource(ctx)
		if sourceErr != nil {
			return fmt.Errorf("read Kubernetes auth JWT: %w", sourceErr)
		}
		if strings.TrimSpace(jwt) == "" {
			return errors.New("kubernetes auth JWT is empty")
		}
		clientToken, lease, err = c.loginKubernetes(ctx, jwt)
	case c.appRoleAuth != nil:
		roleID, sourceErr := c.appRoleAuth.RoleID(ctx)
		if sourceErr != nil {
			return fmt.Errorf("read AppRole role ID: %w", sourceErr)
		}
		if strings.TrimSpace(roleID) == "" {
			return errors.New("AppRole role ID is empty")
		}
		secretID, sourceErr := c.appRoleAuth.SecretID(ctx)
		if sourceErr != nil {
			return fmt.Errorf("read AppRole Secret ID: %w", sourceErr)
		}
		if strings.TrimSpace(secretID) == "" {
			return errors.New("AppRole Secret ID is empty")
		}
		clientToken, lease, err = c.loginAppRole(ctx, roleID, secretID)
	}
	if err != nil {
		return err
	}
	c.token = clientToken
	c.tokenLease = lease
	return nil
}

func (c *Client) usesDynamicAuth() bool {
	return c.kubernetesAuth != nil || c.appRoleAuth != nil
}

func tokenNeedsRefresh(lease tokenLease, now time.Time) bool {
	if lease.expiresAt.IsZero() {
		return false
	}
	if !lease.renewable {
		return !now.Before(lease.expiresAt)
	}
	refreshBefore := min(lease.duration/3, time.Minute)
	remaining := lease.expiresAt.Sub(now)
	return remaining <= refreshBefore
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
