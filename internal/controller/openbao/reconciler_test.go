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
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const (
	testNamespace          = "default"
	testEntityID           = "entity-1"
	testEntityName         = "payments"
	testOwner              = "platform"
	testMetadataKey        = "team"
	testConnectionName     = "openbao"
	testDefaultPolicy      = "default"
	testReacquiredEntityID = "entity-2"
)

func TestEntityReconcilerCreatesAndPersistsIdentity(t *testing.T) {
	connection := readyConnection()
	entity := &openbaov1alpha1.OpenBaoEntity{
		ObjectMeta: metav1.ObjectMeta{Name: testEntityName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntitySpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			Metadata:      map[string]string{testMetadataKey: testOwner},
			Policies:      []string{testDefaultPolicy, testEntityName},
		},
	}
	kubeClient := newTestClient(connection, entity)
	baoClient := &fakeEntityClient{}
	reconciler := &OpenBaoEntityReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (EntityClient, error) {
			return baoClient, nil
		},
	}
	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: entity.Name, Namespace: entity.Namespace}})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 {
		t.Fatalf("RequeueAfter = %s, want periodic drift check", result.RequeueAfter)
	}
	var got openbaov1alpha1.OpenBaoEntity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(entity), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.ID != testEntityID || !conditionTrue(got.Status.Conditions, conditionReady) {
		t.Fatalf("status = %#v, want entity-1 and Ready=True", got.Status)
	}
	if baoClient.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", baoClient.createCalls)
	}

	_, err = reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: entity.Name, Namespace: entity.Namespace}})
	if err != nil {
		t.Fatal(err)
	}
	if baoClient.createCalls != 1 {
		t.Fatalf("create calls after second reconcile = %d, want 1", baoClient.createCalls)
	}
}

func TestEntityReconcilerRefusesUnexpectedAdoption(t *testing.T) {
	connection := readyConnection()
	entity := &openbaov1alpha1.OpenBaoEntity{
		ObjectMeta: metav1.ObjectMeta{Name: testEntityName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntitySpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
		},
	}
	kubeClient := newTestClient(connection, entity)
	baoClient := &fakeEntityClient{entity: &openbaoclient.Entity{ID: testEntityID, Name: testEntityName}}
	reconciler := &OpenBaoEntityReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (EntityClient, error) {
			return baoClient, nil
		},
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: entity.Name, Namespace: entity.Namespace}}); err == nil {
		t.Fatal("expected an error when creation policy refuses adoption")
	}
	var got openbaov1alpha1.OpenBaoEntity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(entity), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions, conditionReady)
	if condition == nil || condition.Reason != "EntityAcquireFailed" || condition.Status != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %#v, want EntityAcquireFailed/False", condition)
	}
	if baoClient.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", baoClient.createCalls)
	}
}

func TestEntityReconcilerPersistsReacquiredID(t *testing.T) {
	connection := readyConnection()
	entity := &openbaov1alpha1.OpenBaoEntity{
		ObjectMeta: metav1.ObjectMeta{Name: testEntityName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntitySpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			Metadata:       map[string]string{testMetadataKey: testOwner},
			Policies:       []string{testDefaultPolicy},
			DeletionPolicy: openbaov1alpha1.DeletionPolicyOrphan,
		},
		Status: openbaov1alpha1.OpenBaoEntityStatus{
			ID:   testEntityID,
			Name: testEntityName,
			Conditions: []metav1.Condition{{
				Type: conditionReady, Status: metav1.ConditionTrue, Reason: reasonReconciled,
			}},
		},
	}
	kubeClient := newTestClient(connection, entity)
	baoClient := &reacquiringEntityClient{}
	reconciler := &OpenBaoEntityReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (EntityClient, error) {
			return baoClient, nil
		},
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: entity.Name, Namespace: entity.Namespace}}); err != nil {
		t.Fatal(err)
	}
	var got openbaov1alpha1.OpenBaoEntity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(entity), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.ID != testReacquiredEntityID {
		t.Fatalf("status ID = %q, want reacquired entity-2", got.Status.ID)
	}
}

func TestConnectionReconcilerRecordsHealthAndAuthentication(t *testing.T) {
	connection := &openbaov1alpha1.OpenBaoConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnectionName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoConnectionSpec{
			Address:        "https://openbao.example.test",
			TokenSecretRef: openbaov1alpha1.SecretKeyReference{Name: "token"},
		},
	}
	kubeClient := newTestClient(connection)
	reconciler := &OpenBaoConnectionReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (ConnectionClient, error) {
			return &fakeConnectionClient{health: &openbaoclient.Health{Version: "2.6.2", Initialized: true}}, nil
		},
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: connection.Name, Namespace: connection.Namespace}}); err != nil {
		t.Fatal(err)
	}
	var got openbaov1alpha1.OpenBaoConnection
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(connection), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Version != "2.6.2" || !got.Status.Authenticated || !conditionTrue(got.Status.Conditions, conditionReady) {
		t.Fatalf("status = %#v, want authenticated v2.6.2 and Ready=True", got.Status)
	}
}

func readyConnection() *openbaov1alpha1.OpenBaoConnection {
	return &openbaov1alpha1.OpenBaoConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnectionName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoConnectionSpec{
			Address:        "https://openbao.example.test",
			TokenSecretRef: openbaov1alpha1.SecretKeyReference{Name: "token"},
		},
		Status: openbaov1alpha1.OpenBaoConnectionStatus{Conditions: []metav1.Condition{{
			Type: conditionReady, Status: metav1.ConditionTrue, Reason: "Reconciled",
		}}},
	}
}

func newTestClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := openbaov1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	for _, object := range objects {
		switch typedObject := object.(type) {
		case *openbaov1alpha1.OpenBaoConnection:
			typedObject.TypeMeta = metav1.TypeMeta{APIVersion: openbaov1alpha1.SchemeGroupVersion.String(), Kind: "OpenBaoConnection"}
		case *openbaov1alpha1.OpenBaoEntity:
			typedObject.TypeMeta = metav1.TypeMeta{APIVersion: openbaov1alpha1.SchemeGroupVersion.String(), Kind: "OpenBaoEntity"}
		case *openbaov1alpha1.OpenBaoEntityAlias:
			typedObject.TypeMeta = metav1.TypeMeta{APIVersion: openbaov1alpha1.SchemeGroupVersion.String(), Kind: "OpenBaoEntityAlias"}
		}
	}
	runtimeObjects := make([]runtime.Object, 0, len(objects))
	for _, object := range objects {
		runtimeObjects = append(runtimeObjects, object)
	}
	result := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&openbaov1alpha1.OpenBaoConnection{}, &openbaov1alpha1.OpenBaoEntity{}, &openbaov1alpha1.OpenBaoEntityAlias{}).
		WithRuntimeObjects(runtimeObjects...).Build()
	return result
}

func conditionTrue(conditions []metav1.Condition, conditionType string) bool {
	condition := findCondition(conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

type fakeConnectionClient struct {
	health *openbaoclient.Health
}

func (f *fakeConnectionClient) CheckHealth(context.Context) (*openbaoclient.Health, error) {
	return f.health, nil
}

func (f *fakeConnectionClient) LookupSelf(context.Context) error { return nil }

type fakeEntityClient struct {
	entity      *openbaoclient.Entity
	createCalls int
}

type reacquiringEntityClient struct {
	created bool
}

func (f *reacquiringEntityClient) GetEntityByID(_ context.Context, id string) (*openbaoclient.Entity, error) {
	if f.created && id == testReacquiredEntityID {
		return &openbaoclient.Entity{
			ID: testReacquiredEntityID, Name: testEntityName,
			Metadata: map[string]string{testMetadataKey: testOwner}, Policies: []string{testDefaultPolicy},
		}, nil
	}
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}

func (reacquiringEntityClient) GetEntityByName(context.Context, string) (*openbaoclient.Entity, error) {
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}

func (f *reacquiringEntityClient) CreateEntity(_ context.Context, request openbaoclient.EntityRequest) (string, error) {
	f.created = true
	return testReacquiredEntityID, nil
}

func (reacquiringEntityClient) UpdateEntity(context.Context, string, openbaoclient.EntityRequest) (*openbaoclient.Entity, error) {
	return nil, fmt.Errorf("unexpected update")
}

func (reacquiringEntityClient) DeleteEntity(context.Context, string) error {
	return fmt.Errorf("unexpected delete")
}

func (f *fakeEntityClient) GetEntityByID(context.Context, string) (*openbaoclient.Entity, error) {
	if f.entity == nil {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	return f.entity, nil
}

func (f *fakeEntityClient) GetEntityByName(context.Context, string) (*openbaoclient.Entity, error) {
	if f.entity == nil {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	return f.entity, nil
}

func (f *fakeEntityClient) CreateEntity(_ context.Context, request openbaoclient.EntityRequest) (string, error) {
	f.createCalls++
	f.entity = &openbaoclient.Entity{ID: testEntityID, Name: request.Name, Metadata: request.Metadata, Policies: request.Policies, Disabled: request.Disabled}
	return f.entity.ID, nil
}

func (f *fakeEntityClient) UpdateEntity(_ context.Context, id string, request openbaoclient.EntityRequest) (*openbaoclient.Entity, error) {
	if f.entity == nil || f.entity.ID != id {
		return nil, fmt.Errorf("entity %s not found", id)
	}
	f.entity.Name = request.Name
	f.entity.Metadata = request.Metadata
	f.entity.Policies = request.Policies
	f.entity.Disabled = request.Disabled
	return f.entity, nil
}

func (f *fakeEntityClient) DeleteEntity(context.Context, string) error { return nil }
