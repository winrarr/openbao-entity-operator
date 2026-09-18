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
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const testMFAMethodName = "duo-login"

func TestMFAMethodReconcilerReactsToSecretRotationWithoutPersistingSecret(t *testing.T) {
	connection := readyConnection()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "duo-secret", Namespace: testNamespace},
		Data:       map[string][]byte{"value": []byte("first-secret")},
	}
	method := &openbaov1alpha1.OpenBaoMFAMethod{
		ObjectMeta: metav1.ObjectMeta{Name: testMFAMethodName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoMFAMethodSpec{
			ConnectionRef:  connectionReference(connection),
			Type:           openbaov1alpha1.OpenBaoMFAMethodTypeDuo,
			MethodID:       testMFAMethodName,
			MethodName:     testMFAMethodName,
			APIHostname:    "api.example.test",
			IntegrationKey: "integration-key",
			SecretKeyRef:   &openbaov1alpha1.SecretKeyReference{Name: secret.Name},
		},
	}
	kubeClient := newTestClient(connection, secret, method)
	baoClient := &fakeMFAMethodClient{}
	reconciler := &OpenBaoMFAMethodReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (MFAMethodClient, error) {
			return baoClient, nil
		},
	}
	request := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(method)}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if baoClient.writeCalls != 1 || baoClient.methods[method.Spec.MethodID]["secret_key"] != "first-secret" {
		t.Fatalf("writes = %d, method = %#v, want first secret", baoClient.writeCalls, baoClient.methods)
	}

	var first openbaov1alpha1.OpenBaoMFAMethod
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(method), &first); err != nil {
		t.Fatal(err)
	}
	firstStatus, err := json.Marshal(first.Status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(firstStatus), "first-secret") {
		t.Fatal("MFA status contains the provider secret")
	}
	firstHash := first.Status.SecretHash

	secret.Data["value"] = []byte("second-secret")
	if err := kubeClient.Update(context.Background(), secret); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if baoClient.writeCalls != 2 || baoClient.methods[method.Spec.MethodID]["secret_key"] != "second-secret" {
		t.Fatalf("writes = %d, method = %#v, want rotated secret", baoClient.writeCalls, baoClient.methods)
	}
	var second openbaov1alpha1.OpenBaoMFAMethod
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(method), &second); err != nil {
		t.Fatal(err)
	}
	if second.Status.SecretHash == firstHash {
		t.Fatal("MFA SecretHash did not change after Secret rotation")
	}
	secondStatus, err := json.Marshal(second.Status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(secondStatus), "second-secret") {
		t.Fatal("rotated MFA status contains the provider secret")
	}
}

type fakeMFAMethodClient struct {
	methods    map[string]openbaoclient.AdditionalObject
	writeCalls int
}

func (f *fakeMFAMethodClient) GetMFAMethod(_ context.Context, _, methodID string) (openbaoclient.AdditionalObject, error) {
	value, ok := f.methods[methodID]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	return cloneAdditionalObject(value), nil
}

func (f *fakeMFAMethodClient) WriteMFAMethod(_ context.Context, _, methodID string, value openbaoclient.AdditionalObject) error {
	if f.methods == nil {
		f.methods = map[string]openbaoclient.AdditionalObject{}
	}
	f.writeCalls++
	f.methods[methodID] = cloneAdditionalObject(value)
	return nil
}

func (f *fakeMFAMethodClient) DeleteMFAMethod(_ context.Context, _, methodID string) error {
	delete(f.methods, methodID)
	return nil
}
