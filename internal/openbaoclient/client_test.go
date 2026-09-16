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
	"net/http/httptest"
	"testing"
	"time"
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
			_, _ = fmt.Fprint(writer, `{"data":{"id":"entity-1","name":"payments","metadata":{},"policies":["default"],"disabled":false}}`)
		case "/v1/identity/entity/id/entity-1":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(writer, `{"data":{"id":"entity-1","name":"payments","metadata":{},"policies":["default"],"disabled":false}}`)
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
	if entity.ID != "entity-1" || entity.Name != "payments" {
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
		_, _ = fmt.Fprint(writer, `{"data":{"id":"entity-1","name":"team/payments"}}`)
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
