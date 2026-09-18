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

package openbao

import (
	"context"
	"maps"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const testLoggerLevel = "debug"

func TestAuthMethodReconcilerCreatesMountAndIsIdempotent(t *testing.T) {
	connection := readyConnection()
	authMethod := &openbaov1alpha1.OpenBaoAuthMethod{
		ObjectMeta: metav1.ObjectMeta{Name: defaultKubernetesAuthRoleMount, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoAuthMethodSpec{OpenBaoMountSpec: openbaov1alpha1.OpenBaoMountSpec{
			ConnectionRef: connectionReference(connection), Path: defaultKubernetesAuthRoleMount, Type: defaultKubernetesAuthRoleMount,
		}},
	}
	kubeClient := newTestClient(connection, authMethod)
	baoClient := &fakeSystemClient{}
	reconciler := &OpenBaoAuthMethodReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error) {
			return baoClient, nil
		},
	}
	request := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(authMethod)}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 {
		t.Fatalf("RequeueAfter = %s, want drift check", result.RequeueAfter)
	}
	if baoClient.authEnableCalls != 1 {
		t.Fatalf("auth enable calls = %d, want 1", baoClient.authEnableCalls)
	}
	var got openbaov1alpha1.OpenBaoAuthMethod
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(authMethod), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Path != defaultKubernetesAuthRoleMount || got.Status.Type != defaultKubernetesAuthRoleMount || !conditionTrue(got.Status.Conditions) {
		t.Fatalf("status = %#v, want reconciled mount", got.Status)
	}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if baoClient.authEnableCalls != 1 {
		t.Fatalf("auth enable calls after second reconcile = %d, want 1", baoClient.authEnableCalls)
	}
}

func TestNamespaceReconcilerWritesMetadata(t *testing.T) {
	connection := readyConnection()
	namespace := &openbaov1alpha1.OpenBaoNamespace{
		ObjectMeta: metav1.ObjectMeta{Name: "platform", Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoNamespaceSpec{
			ConnectionRef: connectionReference(connection), Path: "teams/platform",
			CustomMetadata: map[string]string{"team": "platform"},
		},
	}
	kubeClient := newTestClient(connection, namespace)
	baoClient := &fakeSystemClient{}
	reconciler := &OpenBaoNamespaceReconciler{
		Client:    kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error) { return baoClient, nil },
	}
	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(namespace)}); err != nil {
		t.Fatal(err)
	}
	if baoClient.namespaceWriteCalls != 1 {
		t.Fatalf("namespace write calls = %d, want 1", baoClient.namespaceWriteCalls)
	}
	var got openbaov1alpha1.OpenBaoNamespace
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(namespace), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Path != namespace.Spec.Path || got.Status.ID == "" || !conditionTrue(got.Status.Conditions) {
		t.Fatalf("status = %#v, want reconciled namespace", got.Status)
	}
}

func TestLoggerReconcilerCreatesAndCorrectsLevel(t *testing.T) {
	connection := readyConnection()
	logger := &openbaov1alpha1.OpenBaoLogger{
		ObjectMeta: metav1.ObjectMeta{Name: "audit-logger", Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoLoggerSpec{
			ConnectionRef: connectionReference(connection),
			Name:          "audit",
			Level:         testLoggerLevel,
		},
	}
	kubeClient := newTestClient(connection, logger)
	baoClient := &fakeSystemClient{}
	reconciler := &OpenBaoLoggerReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error) {
			return baoClient, nil
		},
	}
	request := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(logger)}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if baoClient.loggerWriteCalls != 1 || baoClient.loggers["audit"]["level"] != testLoggerLevel {
		t.Fatalf("logger writes = %d, loggers = %#v, want one debug write", baoClient.loggerWriteCalls, baoClient.loggers)
	}
	var got openbaov1alpha1.OpenBaoLogger
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(logger), &got); err != nil {
		t.Fatal(err)
	}
	if !conditionTrue(got.Status.Conditions) || got.Status.Level != testLoggerLevel {
		t.Fatalf("status = %#v, want Ready=True and debug", got.Status)
	}
}

func connectionReference(connection *openbaov1alpha1.OpenBaoConnection) openbaov1alpha1.OpenBaoConnectionReference {
	return openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name}
}

func cloneAdditionalObject(value openbaoclient.AdditionalObject) openbaoclient.AdditionalObject {
	result := make(openbaoclient.AdditionalObject, len(value))
	maps.Copy(result, value)
	return result
}

type fakeSystemClient struct {
	authMethods         map[string]*openbaoclient.Mount
	namespaces          map[string]*openbaoclient.Namespace
	loggers             map[string]openbaoclient.AdditionalObject
	authEnableCalls     int
	namespaceWriteCalls int
	loggerWriteCalls    int
}

func (f *fakeSystemClient) GetMFALoginEnforcement(context.Context, string) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteMFALoginEnforcement(context.Context, string, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) DeleteMFALoginEnforcement(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetCORSConfiguration(context.Context) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteCORSConfiguration(context.Context, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) DeleteCORSConfiguration(context.Context) error { return nil }
func (f *fakeSystemClient) GetAuditRequestHeader(context.Context, string) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteAuditRequestHeader(context.Context, string, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) DeleteAuditRequestHeader(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetUIHeader(context.Context, string) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteUIHeader(context.Context, string, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) DeleteUIHeader(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetRateLimitQuotaConfiguration(context.Context) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteRateLimitQuotaConfiguration(context.Context, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) GetLogger(_ context.Context, name string) (openbaoclient.AdditionalObject, error) {
	value, ok := f.loggers[name]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	return cloneAdditionalObject(value), nil
}

func (f *fakeSystemClient) WriteLogger(_ context.Context, name string, value openbaoclient.AdditionalObject) error {
	if f.loggers == nil {
		f.loggers = map[string]openbaoclient.AdditionalObject{}
	}
	f.loggerWriteCalls++
	f.loggers[name] = cloneAdditionalObject(value)
	return nil
}
func (f *fakeSystemClient) DeleteLogger(_ context.Context, name string) error {
	delete(f.loggers, name)
	return nil
}
func (f *fakeSystemClient) GetEncryptionKeyConfiguration(context.Context) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteEncryptionKeyConfiguration(context.Context, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) GetKeyringRotationConfiguration(context.Context) (openbaoclient.AdditionalObject, error) {
	return nil, nil
}
func (f *fakeSystemClient) WriteKeyringRotationConfiguration(context.Context, openbaoclient.AdditionalObject) error {
	return nil
}
func (f *fakeSystemClient) GetAuthMethod(_ context.Context, path string) (*openbaoclient.Mount, error) {
	value, ok := f.authMethods[path]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	copy := *value
	return &copy, nil
}
func (f *fakeSystemClient) EnableAuthMethod(_ context.Context, path string, request map[string]any) error {
	if f.authMethods == nil {
		f.authMethods = map[string]*openbaoclient.Mount{}
	}
	f.authEnableCalls++
	f.authMethods[path] = &openbaoclient.Mount{Type: request["type"].(string), Accessor: "auth_accessor"}
	return nil
}
func (f *fakeSystemClient) TuneAuthMethod(context.Context, string, map[string]any) error { return nil }
func (f *fakeSystemClient) DisableAuthMethod(_ context.Context, path string) error {
	delete(f.authMethods, path)
	return nil
}
func (f *fakeSystemClient) GetSecretEngine(context.Context, string) (*openbaoclient.Mount, error) {
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}
func (f *fakeSystemClient) EnableSecretEngine(context.Context, string, map[string]any) error {
	return nil
}
func (f *fakeSystemClient) TuneSecretEngine(context.Context, string, map[string]any) error {
	return nil
}
func (f *fakeSystemClient) DisableSecretEngine(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetNamespace(_ context.Context, path string) (*openbaoclient.Namespace, error) {
	value, ok := f.namespaces[path]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	copy := *value
	return &copy, nil
}
func (f *fakeSystemClient) WriteNamespace(_ context.Context, path string, request map[string]any) error {
	if f.namespaces == nil {
		f.namespaces = map[string]*openbaoclient.Namespace{}
	}
	f.namespaceWriteCalls++
	metadata := map[string]any{}
	for key, value := range request["custom_metadata"].(map[string]string) {
		metadata[key] = value
	}
	f.namespaces[path] = &openbaoclient.Namespace{ID: "namespace-1", Path: path, CustomMetadata: metadata}
	return nil
}
func (f *fakeSystemClient) DeleteNamespace(_ context.Context, path string) error {
	delete(f.namespaces, path)
	return nil
}
func (f *fakeSystemClient) GetAuditDevice(context.Context, string) (*openbaoclient.AuditDevice, error) {
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}
func (f *fakeSystemClient) WriteAuditDevice(context.Context, string, map[string]any) error {
	return nil
}
func (f *fakeSystemClient) DeleteAuditDevice(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetRateLimitQuota(context.Context, string) (openbaoclient.RateLimitQuota, error) {
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}
func (f *fakeSystemClient) WriteRateLimitQuota(context.Context, string, openbaoclient.RateLimitQuota) error {
	return nil
}
func (f *fakeSystemClient) DeleteRateLimitQuota(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetWorkflow(context.Context, string) (openbaoclient.Workflow, error) {
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}
func (f *fakeSystemClient) WriteWorkflow(context.Context, string, openbaoclient.Workflow) error {
	return nil
}
func (f *fakeSystemClient) DeleteWorkflow(context.Context, string) error { return nil }
func (f *fakeSystemClient) GetPlugin(context.Context, string, string) (openbaoclient.Plugin, error) {
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}
func (f *fakeSystemClient) WritePlugin(context.Context, string, string, openbaoclient.Plugin) error {
	return nil
}
func (f *fakeSystemClient) DeletePlugin(context.Context, string, string) error { return nil }
