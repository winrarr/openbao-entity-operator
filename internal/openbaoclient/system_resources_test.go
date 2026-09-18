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

const systemResourceTypeField = "type"

func TestSystemResourceClientUsesOpenBaoPaths(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodGet && request.URL.Path == "/v1/sys/audit" {
			_, _ = writer.Write([]byte(`{"data":{"file/":{"type":"file","description":"audit","local":false,"options":{"file_path":"/tmp/audit.log"}}}}`))
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/v1/sys/quotas/rate-limit/payments" {
			_, _ = writer.Write([]byte(`{"data":{"type":"path","rate":10}}`))
			return
		}
		if request.Method == http.MethodGet {
			_, _ = writer.Write([]byte(`{"data":{"type":"kv","path":"payments","custom_metadata":{"team":"platform"},"version":3}}`))
			return
		}
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	apiClient, err := New(server.URL, "test-token", time.Second, nil)
	mustNoError(t, err)
	ctx := context.Background()

	mount, err := apiClient.GetAuthMethod(ctx, "kubernetes")
	mustNoError(t, err)
	mustType(t, "GetAuthMethod", mount.Type, "kv")
	mustNoError(t, apiClient.EnableAuthMethod(ctx, "custom", map[string]any{systemResourceTypeField: "oidc"}))
	mustNoError(t, apiClient.TuneAuthMethod(ctx, "custom", map[string]any{"default_lease_ttl": "1h"}))
	mustNoError(t, apiClient.DisableAuthMethod(ctx, "custom"))
	mount, err = apiClient.GetSecretEngine(ctx, "payments")
	mustNoError(t, err)
	mustType(t, "GetSecretEngine", mount.Type, "kv")
	mustNoError(t, apiClient.EnableSecretEngine(ctx, "archive", map[string]any{systemResourceTypeField: "kv"}))
	mustNoError(t, apiClient.TuneSecretEngine(ctx, "archive", map[string]any{"max_lease_ttl": "2h"}))
	mustNoError(t, apiClient.DisableSecretEngine(ctx, "archive"))
	namespace, err := apiClient.GetNamespace(ctx, "teams/platform")
	mustNoError(t, err)
	mustString(t, "GetNamespace", namespace.Path, "payments")
	mustNoError(t, apiClient.WriteNamespace(ctx, "teams/platform", map[string]any{"custom_metadata": map[string]string{"team": "platform"}}))
	mustNoError(t, apiClient.DeleteNamespace(ctx, "teams/platform"))
	device, err := apiClient.GetAuditDevice(ctx, "file")
	mustNoError(t, err)
	mustType(t, "GetAuditDevice", device.Type, "file")
	mustNoError(t, apiClient.WriteAuditDevice(ctx, "file", map[string]any{systemResourceTypeField: "file"}))
	mustNoError(t, apiClient.DeleteAuditDevice(ctx, "file"))
	quota, err := apiClient.GetRateLimitQuota(ctx, "payments")
	mustNoError(t, err)
	mustString(t, "GetRateLimitQuota", quota[systemResourceTypeField].(string), "path")
	mustNoError(t, apiClient.WriteRateLimitQuota(ctx, "payments", RateLimitQuota{systemResourceTypeField: "path", "rate": 10.0}))
	mustNoError(t, apiClient.DeleteRateLimitQuota(ctx, "payments"))
	workflow, err := apiClient.GetWorkflow(ctx, "payments/rotation")
	mustNoError(t, err)
	mustFloat(t, "GetWorkflow", workflow["version"].(float64), 3)
	mustNoError(t, apiClient.WriteWorkflow(ctx, "payments/rotation", Workflow{"workflow": "{}"}))
	mustNoError(t, apiClient.DeleteWorkflow(ctx, "payments/rotation"))
	plugin, err := apiClient.GetPlugin(ctx, "secret", "my-plugin")
	mustNoError(t, err)
	mustType(t, "GetPlugin", plugin[systemResourceTypeField].(string), "kv")
	mustNoError(t, apiClient.WritePlugin(ctx, "secret", "my-plugin", Plugin{systemResourceTypeField: "secret"}))
	mustNoError(t, apiClient.DeletePlugin(ctx, "secret", "my-plugin"))

	want := []string{
		"GET /v1/sys/auth/kubernetes",
		"POST /v1/sys/auth/custom",
		"POST /v1/sys/auth/custom/tune",
		"DELETE /v1/sys/auth/custom",
		"GET /v1/sys/mounts/payments",
		"POST /v1/sys/mounts/archive",
		"POST /v1/sys/mounts/archive/tune",
		"DELETE /v1/sys/mounts/archive",
		"GET /v1/sys/namespaces/teams/platform",
		"POST /v1/sys/namespaces/teams/platform",
		"DELETE /v1/sys/namespaces/teams/platform",
		"GET /v1/sys/audit",
		"POST /v1/sys/audit/file",
		"DELETE /v1/sys/audit/file",
		"GET /v1/sys/quotas/rate-limit/payments",
		"POST /v1/sys/quotas/rate-limit/payments",
		"DELETE /v1/sys/quotas/rate-limit/payments",
		"GET /v1/sys/workflows/manage/payments/rotation",
		"POST /v1/sys/workflows/manage/payments/rotation",
		"DELETE /v1/sys/workflows/manage/payments/rotation",
		"GET /v1/sys/plugins/catalog/secret/my-plugin",
		"POST /v1/sys/plugins/catalog/secret/my-plugin",
		"DELETE /v1/sys/plugins/catalog/secret/my-plugin",
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

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustType(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s type = %q, want %q", name, got, want)
	}
}

func mustString(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s value = %q, want %q", name, got, want)
	}
}

func mustFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if got != want {
		t.Fatalf("%s value = %v, want %v", name, got, want)
	}
}
