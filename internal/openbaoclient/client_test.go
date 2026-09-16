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
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	clientEntityID        = "entity-1"
	clientUpdatedEntityID = "entity-2"
	clientAliasID         = "alias-1"
	clientAliasName       = "payments-login"
	clientMountAccessor   = "auth_kubernetes_123"
	clientGroupID         = "group-1"
	clientGroupName       = "platform"
	clientPolicyName      = "payments"
	clientPolicyRules     = "path \"identity/*\" { capabilities = [\"read\"] }"
	testAuthRole          = "operator"
	testJWT               = "jwt-1"
	testLookupSelfPath    = "/v1/auth/token/lookup-self"
	testClientToken       = "client-token"
)

func TestEntityClientUsesOpenBaoHeadersAndPaths(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Vault-Token") != "test-token" {
			t.Errorf("X-Vault-Token = %q, want test-token", request.Header.Get("X-Vault-Token"))
		}
		if request.Header.Get("X-Vault-Request") != "true" {
			t.Errorf("X-Vault-Request = %q, want true", request.Header.Get("X-Vault-Request"))
		}
		if request.Header.Get("X-Vault-Namespace") != "" {
			t.Errorf("X-Vault-Namespace = %q, want empty for root namespace", request.Header.Get("X-Vault-Namespace"))
		}
		requests = append(requests, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/v1/identity/entity/name/payments":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"payments","metadata":{},"policies":["default"],"disabled":false}}`, clientEntityID)
		case "/v1/identity/entity/id/" + clientEntityID:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"payments","metadata":{},"policies":["default"],"disabled":false}}`, clientEntityID)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	entity, err := apiClient.GetEntityByName(context.Background(), "payments")
	if err != nil {
		t.Fatal(err)
	}
	if entity.ID != clientEntityID || entity.Name != "payments" {
		t.Fatalf("entity = %#v, want entity-1/payments", entity)
	}
	if got, want := requests, []string{"GET /v1/identity/entity/name/payments"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestKubernetesAuthClientLogsInAndRenewsToken(t *testing.T) {
	var loginCalls, lookupCalls, renewCalls, jwtCalls int
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/custom-kubernetes/login":
			loginCalls++
			var body map[string]string
			decodeRequestBody(t, request, &body)
			if body["jwt"] != testJWT || body["role"] != testAuthRole {
				t.Fatalf("login body = %#v, want jwt-1/operator", body)
			}
			_, _ = fmt.Fprintf(writer, `{"auth":{"client_token":%q,"lease_duration":300,"renewable":true}}`, testClientToken)
		case testLookupSelfPath:
			lookupCalls++
			if got, want := request.Header.Get("X-Vault-Token"), testClientToken; got != want {
				t.Fatalf("lookup token = %q, want %q", got, want)
			}
			_, _ = fmt.Fprint(writer, `{}`)
		case "/v1/auth/token/renew-self":
			renewCalls++
			if got, want := request.Header.Get("X-Vault-Token"), testClientToken; got != want {
				t.Fatalf("renew token = %q, want %q", got, want)
			}
			var body map[string]string
			decodeRequestBody(t, request, &body)
			if body["increment"] != "" {
				t.Fatalf("renew body = %#v, want empty increment", body)
			}
			_, _ = fmt.Fprint(writer, `{"auth":{"lease_duration":300,"renewable":true}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := NewWithKubernetesAuth(server.URL, KubernetesAuthOptions{
		MountPath: "custom-kubernetes",
		Role:      testAuthRole,
		JWTSource: func(context.Context) (string, error) {
			jwtCalls++
			return testJWT, nil
		},
	}, time.Second, nil, "platform/production")
	if err != nil {
		t.Fatal(err)
	}
	apiClient.now = func() time.Time { return now }

	if err := apiClient.LookupSelf(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(241 * time.Second)
	if err := apiClient.LookupSelf(context.Background()); err != nil {
		t.Fatal(err)
	}
	if loginCalls != 1 || lookupCalls != 2 || renewCalls != 1 || jwtCalls != 1 {
		t.Fatalf("login/lookup/renew/JWT calls = %d/%d/%d/%d, want 1/2/1/1", loginCalls, lookupCalls, renewCalls, jwtCalls)
	}
}

func TestKubernetesAuthClientReloginsAfterUnauthorizedResponse(t *testing.T) {
	var loginCalls, lookupCalls, jwtCalls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/kubernetes/login":
			loginCalls++
			_, _ = fmt.Fprintf(writer, `{"auth":{"client_token":"client-token-%d","renewable":false}}`, loginCalls)
		case testLookupSelfPath:
			lookupCalls++
			if request.Header.Get("X-Vault-Token") == "client-token-1" {
				http.Error(writer, "token expired", http.StatusForbidden)
				return
			}
			_, _ = fmt.Fprint(writer, `{}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := NewWithKubernetesAuth(server.URL, KubernetesAuthOptions{
		Role: testAuthRole,
		JWTSource: func(context.Context) (string, error) {
			jwtCalls++
			return fmt.Sprintf("jwt-%d", jwtCalls), nil
		},
	}, time.Second, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := apiClient.LookupSelf(context.Background()); err != nil {
		t.Fatal(err)
	}
	if loginCalls != 2 || lookupCalls != 2 || jwtCalls != 2 {
		t.Fatalf("login/lookup/JWT calls = %d/%d/%d, want 2/2/2", loginCalls, lookupCalls, jwtCalls)
	}
}

func TestKubernetesAuthClientUsesCABundle(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/kubernetes/login":
			_, _ = fmt.Fprint(writer, `{"auth":{"client_token":"client-token","renewable":false}}`)
		case testLookupSelfPath:
			if got, want := request.Header.Get("X-Vault-Token"), testClientToken; got != want {
				t.Fatalf("lookup token = %q, want %q", got, want)
			}
			_, _ = fmt.Fprint(writer, `{}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	caBundle := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	apiClient, err := NewWithKubernetesAuth(server.URL, KubernetesAuthOptions{
		Role: testAuthRole,
		JWTSource: func(context.Context) (string, error) {
			return testJWT, nil
		},
	}, time.Second, caBundle, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := apiClient.LookupSelf(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewWithKubernetesAuthRejectsInvalidOptions(t *testing.T) {
	jwtSource := func(context.Context) (string, error) { return "jwt", nil }
	for name, options := range map[string]KubernetesAuthOptions{
		"empty role":       {JWTSource: jwtSource},
		"role whitespace":  {Role: "operator role", JWTSource: jwtSource},
		"nil JWT source":   {Role: testAuthRole},
		"auth prefix":      {Role: testAuthRole, MountPath: "auth/kubernetes", JWTSource: jwtSource},
		"empty mount path": {Role: testAuthRole, MountPath: "custom//mount", JWTSource: jwtSource},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewWithKubernetesAuth("http://openbao.example.test", options, time.Second, nil, ""); err == nil {
				t.Fatal("NewWithKubernetesAuth returned nil error")
			}
		})
	}
}

func TestClientUsesNamespaceHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got, want := request.Header.Get("X-Vault-Namespace"), "platform/production"; got != want {
			t.Fatalf("X-Vault-Namespace = %q, want %q", got, want)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"data":{"id":"entity-1","name":"payments"}}`)
	}))
	defer server.Close()

	apiClient, err := NewWithNamespace(server.URL, "test-token", time.Second, nil, "platform/production")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.GetEntityByName(context.Background(), "payments"); err != nil {
		t.Fatal(err)
	}
}

func TestNewWithNamespaceRejectsInvalidNamespace(t *testing.T) {
	for _, namespace := range []string{"/platform", "platform/", "platform//production", "platform production", "identity"} {
		t.Run(namespace, func(t *testing.T) {
			if _, err := NewWithNamespace("http://openbao.example.test", "test-token", time.Second, nil, namespace); err == nil {
				t.Fatalf("NewWithNamespace(%q) returned nil error", namespace)
			}
		})
	}
}

func TestCheckHealthAcceptsSealedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/sys/health" {
			t.Fatalf("path = %q, want /v1/sys/health", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(writer, `{"version":"2.6.2","initialized":true,"sealed":true,"standby":false}`)
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	health, err := apiClient.CheckHealth(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !health.Sealed || health.Version != "2.6.2" {
		t.Fatalf("health = %#v, want sealed v2.6.2", health)
	}
}

func TestEntityClientEscapesPathParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/v1/identity/entity/name/team%2Fpayments" {
			t.Fatalf("escaped path = %q, want encoded entity name", request.URL.EscapedPath())
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"team/payments"}}`, clientEntityID)
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.GetEntityByName(context.Background(), "team/payments"); err != nil {
		t.Fatal(err)
	}
}

func TestEntityAliasClientUsesOpenBaoAliasEndpoints(t *testing.T) {
	var requestBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/v1/identity/entity-alias/id" && request.URL.Query().Get("list") == "true":
			_, _ = fmt.Fprint(writer, `{"data":{"keys":["alias-1"]}}`)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/identity/entity-alias/id/"+clientAliasID:
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"%s","mount_accessor":"%s","canonical_id":"%s"}}`, clientAliasID, clientAliasName, clientMountAccessor, clientEntityID)
		case request.Method == http.MethodPost && request.URL.Path == "/v1/identity/entity-alias":
			decodeRequestBody(t, request, &requestBody)
			_, _ = fmt.Fprint(writer, `{"data":{"id":"alias-1"}}`)
		case request.Method == http.MethodPost && request.URL.Path == "/v1/identity/entity-alias/id/"+clientAliasID:
			decodeRequestBody(t, request, &requestBody)
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"%s","mount_accessor":"%s","canonical_id":"%s"}}`, clientAliasID, clientAliasName, clientMountAccessor, clientUpdatedEntityID)
		case request.Method == http.MethodDelete && request.URL.Path == "/v1/identity/entity-alias/id/"+clientAliasID:
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := apiClient.ListEntityAliasIDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(ids) != "[alias-1]" {
		t.Fatalf("alias IDs = %v, want [alias-1]", ids)
	}
	alias, err := apiClient.GetEntityAliasByID(context.Background(), "alias-1")
	if err != nil {
		t.Fatal(err)
	}
	if alias.CanonicalID != clientEntityID || alias.MountAccessor != clientMountAccessor {
		t.Fatalf("alias = %#v, want entity-1/auth_kubernetes_123", alias)
	}
	if _, err := apiClient.CreateEntityAlias(context.Background(), EntityAliasRequest{
		Name: clientAliasName, MountAccessor: clientMountAccessor, CanonicalID: clientEntityID,
	}); err != nil {
		t.Fatal(err)
	}
	if requestBody["canonical_id"] != clientEntityID || requestBody["name"] != clientAliasName {
		t.Fatalf("create body = %#v, want canonical_id/name", requestBody)
	}
	updated, err := apiClient.UpdateEntityAlias(context.Background(), clientAliasID, EntityAliasRequest{CanonicalID: clientUpdatedEntityID})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CanonicalID != clientUpdatedEntityID || requestBody["canonical_id"] != clientUpdatedEntityID {
		t.Fatalf("updated alias/body = %#v/%#v, want entity-2", updated, requestBody)
	}
	if err := apiClient.DeleteEntityAlias(context.Background(), clientAliasID); err != nil {
		t.Fatal(err)
	}
}

func TestGroupClientUsesOpenBaoGroupEndpoints(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/v1/identity/group/name/"+clientGroupName:
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"%s","type":"internal","metadata":{"team":"platform"},"policies":["default"],"member_entity_ids":["entity-1"],"member_group_ids":[]}}`, clientGroupID, clientGroupName)
		case request.Method == http.MethodPost && request.URL.Path == "/v1/identity/group":
			decodeAnyRequestBody(t, request, &requestBody)
			_, _ = fmt.Fprint(writer, `{"data":{"id":"group-1"}}`)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/identity/group/id/"+clientGroupID:
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s","name":"%s","type":"internal","metadata":{"team":"platform"},"policies":["default"],"member_entity_ids":["entity-1"],"member_group_ids":[]}}`, clientGroupID, clientGroupName)
		case request.Method == http.MethodPost && request.URL.Path == "/v1/identity/group/id/"+clientGroupID:
			decodeAnyRequestBody(t, request, &requestBody)
			_, _ = fmt.Fprintf(writer, `{"data":{"id":"%s"}}`, clientGroupID)
		case request.Method == http.MethodDelete && request.URL.Path == "/v1/identity/group/id/"+clientGroupID:
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	group, err := apiClient.GetGroupByName(context.Background(), clientGroupName)
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != clientGroupID || group.MemberEntityIDs[0] != "entity-1" {
		t.Fatalf("group = %#v, want group-1 with entity-1", group)
	}
	if _, err := apiClient.CreateGroup(context.Background(), GroupRequest{
		Name: clientGroupName, Type: "internal", Policies: []string{"default"}, MemberEntityIDs: []string{"entity-1"}, MemberGroupIDs: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	if requestBody["name"] != clientGroupName || requestBody["type"] != "internal" {
		t.Fatalf("create body = %#v, want group name/type", requestBody)
	}
	updated, err := apiClient.UpdateGroup(context.Background(), clientGroupID, GroupRequest{MemberEntityIDs: []string{"entity-2"}, MemberGroupIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != clientGroupID || requestBody["member_entity_ids"].([]any)[0] != "entity-2" {
		t.Fatalf("updated group/body = %#v/%#v, want group-1/entity-2", updated, requestBody)
	}
	if err := apiClient.DeleteGroup(context.Background(), clientGroupID); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyClientUsesOpenBaoPolicyEndpoints(t *testing.T) {
	var requestBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/sys/policies/acl/"+clientPolicyName {
			t.Fatalf("path = %q, want policy endpoint", request.URL.Path)
		}
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"data":{"name":%q,"policy":%q,"version":2}}`, clientPolicyName, clientPolicyRules)
		case http.MethodPost:
			decodeRequestBody(t, request, &requestBody)
			writer.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := apiClient.GetPolicy(context.Background(), clientPolicyName)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Name != clientPolicyName || policy.Rules != clientPolicyRules || policy.Version != 2 {
		t.Fatalf("policy = %#v, want name/rules/version", policy)
	}
	if err := apiClient.WritePolicy(context.Background(), clientPolicyName, PolicyRequest{Rules: clientPolicyRules}); err != nil {
		t.Fatal(err)
	}
	if requestBody["policy"] != clientPolicyRules {
		t.Fatalf("write body = %#v, want policy document", requestBody)
	}
	if err := apiClient.DeletePolicy(context.Background(), clientPolicyName); err != nil {
		t.Fatal(err)
	}
}

func decodeRequestBody(t *testing.T, request *http.Request, target *map[string]string) {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatal(err)
	}
}

func decodeAnyRequestBody(t *testing.T, request *http.Request, target *map[string]any) {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatal(err)
	}
}
