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
	"net/http/httptest"
	"testing"
	"time"
)

func TestConfigurationClientsUseTypedOpenBaoPaths(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method+" "+request.URL.RequestURI())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/identity/group-alias":
			_, _ = writer.Write([]byte(`{"data":{"id":"alias-1"}}`))
		case "/v1/identity/group-alias/id":
			_, _ = writer.Write([]byte(`{"data":{"keys":["alias-1"]}}`))
		case "/v1/sys/policies/password/enterprise":
			_, _ = writer.Write([]byte(`{"data":{"policy":"length = 20"}}`))
		default:
			_, _ = writer.Write([]byte(`{"data":{"id":"alias-1","name":"payments","mount_accessor":"auth_oidc","canonical_id":"group-1"}}`))
		}
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	mustNoError(t, err)
	ctx := context.Background()

	_, err = apiClient.GetAppRole(ctx, "approle", "payments")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteAppRole(ctx, "approle", "payments", AppRoleRequest{TokenType: "service"}))
	mustNoError(t, apiClient.DeleteAppRole(ctx, "approle", "payments"))
	_, err = apiClient.GetTokenRole(ctx, "payments")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteTokenRole(ctx, "payments", TokenRoleRequest{TokenType: "service"}))
	mustNoError(t, apiClient.DeleteTokenRole(ctx, "payments"))
	policy, err := apiClient.GetPasswordPolicy(ctx, "enterprise")
	mustNoError(t, err)
	mustString(t, "password policy", policy.Rules, "length = 20")
	mustNoError(t, apiClient.WritePasswordPolicy(ctx, "enterprise", "length = 20"))
	mustNoError(t, apiClient.DeletePasswordPolicy(ctx, "enterprise"))
	_, err = apiClient.GetOIDCConfig(ctx)
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteOIDCConfig(ctx, OIDCObject{"issuer": "https://openbao.example.test"}))
	_, err = apiClient.GetOIDCResource(ctx, oidcProviderSegment, "provider")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteOIDCResource(ctx, oidcProviderSegment, "provider", OIDCObject{"issuer": "https://openbao.example.test"}))
	mustNoError(t, apiClient.DeleteOIDCResource(ctx, oidcProviderSegment, "provider"))
	_, err = apiClient.GetGroupAliasByID(ctx, "alias-1")
	mustNoError(t, err)
	_, err = apiClient.ListGroupAliasIDs(ctx)
	mustNoError(t, err)
	_, err = apiClient.CreateGroupAlias(ctx, GroupAliasRequest{Name: clientPolicyName})
	mustNoError(t, err)
	_, err = apiClient.UpdateGroupAlias(ctx, "alias-1", GroupAliasRequest{Name: clientPolicyName})
	mustNoError(t, err)
	mustNoError(t, apiClient.DeleteGroupAlias(ctx, "alias-1"))

	want := []string{
		"GET /v1/auth/approle/role/payments",
		"POST /v1/auth/approle/role/payments",
		"DELETE /v1/auth/approle/role/payments",
		"GET /v1/auth/token/roles/payments",
		"POST /v1/auth/token/roles/payments",
		"DELETE /v1/auth/token/roles/payments",
		"GET /v1/sys/policies/password/enterprise",
		"POST /v1/sys/policies/password/enterprise",
		"DELETE /v1/sys/policies/password/enterprise",
		"GET /v1/identity/oidc/config",
		"POST /v1/identity/oidc/config",
		"GET /v1/identity/oidc/provider/provider",
		"POST /v1/identity/oidc/provider/provider",
		"DELETE /v1/identity/oidc/provider/provider",
		"GET /v1/identity/group-alias/id/alias-1",
		"GET /v1/identity/group-alias/id?list=true",
		"POST /v1/identity/group-alias",
		"POST /v1/identity/group-alias/id/alias-1",
		"DELETE /v1/identity/group-alias/id/alias-1",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("request[%d] = %q, want %q", i, requests[i], want[i])
		}
	}
}

func TestAdditionalConfigurationClientsUseOpenBaoPaths(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method+" "+request.URL.RequestURI())
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodGet && request.URL.Path == "/v1/identity/persona/id" {
			_, _ = writer.Write([]byte(`{"data":{"keys":["persona-1"]}}`))
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/v1/sys/loggers/audit" {
			_, _ = writer.Write([]byte(`{"data":{"audit":"debug"}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"data":{"id":"resource-1","level":"info","method_name":"login"}}`))
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	mustNoError(t, err)
	ctx := context.Background()

	_, err = apiClient.ListPersonaIDs(ctx)
	mustNoError(t, err)
	_, err = apiClient.GetPersonaByID(ctx, "persona-1")
	mustNoError(t, err)
	_, err = apiClient.CreatePersona(ctx, PersonaRequest{Name: "payments"})
	mustNoError(t, err)
	_, err = apiClient.UpdatePersona(ctx, "persona-1", PersonaRequest{Name: "payments"})
	mustNoError(t, err)
	mustNoError(t, apiClient.DeletePersona(ctx, "persona-1"))

	_, err = apiClient.GetMFALoginEnforcement(ctx, "payments")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteMFALoginEnforcement(ctx, "payments", AdditionalObject{"mfa_method_ids": []string{"mfa-1"}}))
	mustNoError(t, apiClient.DeleteMFALoginEnforcement(ctx, "payments"))
	_, err = apiClient.GetMFAMethod(ctx, "totp", "mfa-1")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteMFAMethod(ctx, "totp", "mfa-1", AdditionalObject{"method_name": "login"}))
	mustNoError(t, apiClient.DeleteMFAMethod(ctx, "totp", "mfa-1"))

	_, err = apiClient.GetCORSConfiguration(ctx)
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteCORSConfiguration(ctx, AdditionalObject{"enable": true}))
	mustNoError(t, apiClient.DeleteCORSConfiguration(ctx))
	_, err = apiClient.GetAuditRequestHeader(ctx, "X-Request-ID")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteAuditRequestHeader(ctx, "X-Request-ID", AdditionalObject{"hmac": true}))
	mustNoError(t, apiClient.DeleteAuditRequestHeader(ctx, "X-Request-ID"))
	_, err = apiClient.GetUIHeader(ctx, "X-Frame-Options")
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteUIHeader(ctx, "X-Frame-Options", AdditionalObject{"values": []string{"DENY"}}))
	mustNoError(t, apiClient.DeleteUIHeader(ctx, "X-Frame-Options"))
	_, err = apiClient.GetRateLimitQuotaConfiguration(ctx)
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteRateLimitQuotaConfiguration(ctx, AdditionalObject{"enable_rate_limit_audit_logging": true}))
	logger, err := apiClient.GetLogger(ctx, "audit")
	mustNoError(t, err)
	mustString(t, "GetLogger", logger["level"].(string), "debug")
	mustNoError(t, apiClient.WriteLogger(ctx, "audit", AdditionalObject{"level": "info"}))
	mustNoError(t, apiClient.DeleteLogger(ctx, "audit"))
	_, err = apiClient.GetEncryptionKeyConfiguration(ctx)
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteEncryptionKeyConfiguration(ctx, AdditionalObject{"enabled": true}))
	_, err = apiClient.GetKeyringRotationConfiguration(ctx)
	mustNoError(t, err)
	mustNoError(t, apiClient.WriteKeyringRotationConfiguration(ctx, AdditionalObject{"enabled": true}))

	want := []string{
		"GET /v1/identity/persona/id?list=true",
		"GET /v1/identity/persona/id/persona-1",
		"POST /v1/identity/persona",
		"POST /v1/identity/persona/id/persona-1",
		"DELETE /v1/identity/persona/id/persona-1",
		"GET /v1/identity/mfa/login-enforcement/payments",
		"POST /v1/identity/mfa/login-enforcement/payments",
		"DELETE /v1/identity/mfa/login-enforcement/payments",
		"GET /v1/identity/mfa/method/totp/mfa-1",
		"POST /v1/identity/mfa/method/totp/mfa-1",
		"DELETE /v1/identity/mfa/method/totp/mfa-1",
		"GET /v1/sys/config/cors",
		"POST /v1/sys/config/cors",
		"DELETE /v1/sys/config/cors",
		"GET /v1/sys/config/auditing/request-headers/X-Request-ID",
		"POST /v1/sys/config/auditing/request-headers/X-Request-ID",
		"DELETE /v1/sys/config/auditing/request-headers/X-Request-ID",
		"GET /v1/sys/config/ui/headers/X-Frame-Options",
		"POST /v1/sys/config/ui/headers/X-Frame-Options",
		"DELETE /v1/sys/config/ui/headers/X-Frame-Options",
		"GET /v1/sys/quotas/config",
		"POST /v1/sys/quotas/config",
		"GET /v1/sys/loggers/audit",
		"POST /v1/sys/loggers/audit",
		"DELETE /v1/sys/loggers/audit",
		"GET /v1/sys/rotate/config",
		"POST /v1/sys/rotate/config",
		"GET /v1/sys/rotate/keyring/config",
		"POST /v1/sys/rotate/keyring/config",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("request[%d] = %q, want %q", i, requests[i], want[i])
		}
	}
}
