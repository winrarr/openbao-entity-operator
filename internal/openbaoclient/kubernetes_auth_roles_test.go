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

func TestKubernetesAuthRoleClientUsesRolePathsAndWireFields(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			_, _ = writer.Write([]byte(`{"data":{"bound_service_account_names":["payments"],"bound_service_account_namespaces":["default"],"token_policies":["payments"],"token_ttl":3600,"token_max_ttl":7200,"token_period":0}}`))
		case http.MethodPost:
			var body map[string]any
			decodeAnyRequestBody(t, request, &body)
			if body["bound_service_account_names"].([]any)[0] != clientPolicyName || body["token_policies"].([]any)[0] != clientPolicyName || body["token_ttl"].(float64) != 3600 || body["token_max_ttl"].(float64) != 7200 {
				t.Fatalf("role request = %#v, want token-prefixed role configuration", body)
			}
			_, _ = writer.Write([]byte(`{}`))
		case http.MethodDelete:
			_, _ = writer.Write([]byte(`{}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	role, err := apiClient.GetKubernetesAuthRole(context.Background(), "custom-kubernetes", clientPolicyName)
	if err != nil {
		t.Fatal(err)
	}
	if role.TokenTTL != 3600 || role.TokenMaxTTL != 7200 || role.EffectivePolicies()[0] != clientPolicyName {
		t.Fatalf("role = %#v, want OpenBao Kubernetes Auth role", role)
	}
	if err := apiClient.WriteKubernetesAuthRole(context.Background(), "custom-kubernetes", clientPolicyName, KubernetesAuthRoleRequest{
		BoundServiceAccountNames:      []string{clientPolicyName},
		BoundServiceAccountNamespaces: []string{clientDefaultPolicy},
		TokenPolicies:                 []string{clientPolicyName},
		TokenTTL:                      3600,
		TokenMaxTTL:                   7200,
	}); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteKubernetesAuthRole(context.Background(), "custom-kubernetes", clientPolicyName); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET /v1/auth/custom-kubernetes/role/" + clientPolicyName,
		"POST /v1/auth/custom-kubernetes/role/" + clientPolicyName,
		"DELETE /v1/auth/custom-kubernetes/role/" + clientPolicyName,
	}
	if got := requests; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestKubernetesAuthRoleEffectivePoliciesSupportsAliasResponse(t *testing.T) {
	role := &KubernetesAuthRole{Policies: []string{clientDefaultPolicy, clientPolicyName}}
	if got := role.EffectivePolicies(); len(got) != 2 || got[0] != clientDefaultPolicy || got[1] != clientPolicyName {
		t.Fatalf("effective policies = %v, want alias response policies", got)
	}
}
