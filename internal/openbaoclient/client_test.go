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
