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
	"slices"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const (
	testAliasName       = "payments-login"
	testMountAccessor   = "auth_kubernetes_123"
	testAliasID         = "alias-1"
	testAliasEntityName = "payments"
)

func TestEntityAliasReconcilerCreatesAndPersistsAlias(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	alias := &openbaov1alpha1.OpenBaoEntityAlias{
		ObjectMeta: metav1.ObjectMeta{Name: testAliasName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntityAliasSpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			EntityRef:     openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
			MountAccessor: testMountAccessor,
			Name:          testAliasName,
		},
	}
	kubeClient := newTestClient(connection, entity, alias)
	baoClient := &fakeAliasClient{}
	reconciler := &OpenBaoEntityAliasReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (AliasClient, error) {
			return baoClient, nil
		},
	}

	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: alias.Name, Namespace: alias.Namespace}})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 {
		t.Fatalf("RequeueAfter = %s, want periodic drift check", result.RequeueAfter)
	}
	var got openbaov1alpha1.OpenBaoEntityAlias
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(alias), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.ID != testAliasID || got.Status.CanonicalID != testEntityID || !conditionTrue(got.Status.Conditions) {
		t.Fatalf("status = %#v, want alias-1, entity-1, and Ready=True", got.Status)
	}
	if baoClient.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", baoClient.createCalls)
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: alias.Name, Namespace: alias.Namespace}}); err != nil {
		t.Fatal(err)
	}
	if baoClient.createCalls != 1 || baoClient.updateCalls != 0 {
		t.Fatalf("calls after second reconcile = create %d/update %d, want 1/0", baoClient.createCalls, baoClient.updateCalls)
	}
}

func TestEntityAliasReconcilerAdoptsExistingAlias(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	alias := &openbaov1alpha1.OpenBaoEntityAlias{
		ObjectMeta: metav1.ObjectMeta{Name: testAliasName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntityAliasSpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			EntityRef:      openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
			MountAccessor:  testMountAccessor,
			Name:           testAliasName,
			CreationPolicy: openbaov1alpha1.CreationPolicyAdopt,
		},
	}
	kubeClient := newTestClient(connection, entity, alias)
	baoClient := &fakeAliasClient{aliases: map[string]*openbaoclient.EntityAlias{
		testAliasID: {ID: testAliasID, Name: testAliasName, MountAccessor: testMountAccessor, CanonicalID: testEntityID},
	}}
	reconciler := newAliasReconciler(kubeClient, baoClient)

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: alias.Name, Namespace: alias.Namespace}}); err != nil {
		t.Fatal(err)
	}
	var got openbaov1alpha1.OpenBaoEntityAlias
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(alias), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.ID != testAliasID || baoClient.createCalls != 0 {
		t.Fatalf("status ID/create calls = %q/%d, want alias-1/0", got.Status.ID, baoClient.createCalls)
	}
}

func TestEntityAliasReconcilerCorrectsCanonicalEntityDrift(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	alias := &openbaov1alpha1.OpenBaoEntityAlias{
		ObjectMeta: metav1.ObjectMeta{Name: testAliasName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntityAliasSpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			EntityRef:     openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
			MountAccessor: testMountAccessor,
			Name:          testAliasName,
		},
		Status: openbaov1alpha1.OpenBaoEntityAliasStatus{ID: testAliasID},
	}
	kubeClient := newTestClient(connection, entity, alias)
	baoClient := &fakeAliasClient{aliases: map[string]*openbaoclient.EntityAlias{
		testAliasID: {ID: testAliasID, Name: testAliasName, MountAccessor: testMountAccessor, CanonicalID: "old-entity"},
	}}
	reconciler := newAliasReconciler(kubeClient, baoClient)

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: alias.Name, Namespace: alias.Namespace}}); err != nil {
		t.Fatal(err)
	}
	if baoClient.updateCalls != 1 || baoClient.aliases[testAliasID].CanonicalID != testEntityID {
		t.Fatalf("update calls/canonical ID = %d/%q, want 1/%q", baoClient.updateCalls, baoClient.aliases[testAliasID].CanonicalID, testEntityID)
	}
}

func TestEntityAliasReconcilerRefusesUnexpectedAdoption(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	alias := &openbaov1alpha1.OpenBaoEntityAlias{
		ObjectMeta: metav1.ObjectMeta{Name: testAliasName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntityAliasSpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			EntityRef:     openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
			MountAccessor: testMountAccessor,
			Name:          testAliasName,
		},
	}
	kubeClient := newTestClient(connection, entity, alias)
	baoClient := &fakeAliasClient{aliases: map[string]*openbaoclient.EntityAlias{
		testAliasID: {ID: testAliasID, Name: testAliasName, MountAccessor: testMountAccessor, CanonicalID: testEntityID},
	}}
	reconciler := newAliasReconciler(kubeClient, baoClient)

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: alias.Name, Namespace: alias.Namespace}}); err == nil {
		t.Fatal("expected an error when creation policy refuses adoption")
	}
	var got openbaov1alpha1.OpenBaoEntityAlias
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(alias), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions)
	if condition == nil || condition.Reason != "AliasAcquireFailed" || condition.Status != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %#v, want AliasAcquireFailed/False", condition)
	}
}

func TestEntityAliasReconcilerDeletesAlias(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	alias := &openbaov1alpha1.OpenBaoEntityAlias{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testAliasName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoEntityAliasSpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			EntityRef:      openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
			MountAccessor:  testMountAccessor,
			Name:           testAliasName,
			DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
		},
		Status: openbaov1alpha1.OpenBaoEntityAliasStatus{ID: testAliasID},
	}
	kubeClient := newTestClient(connection, entity, alias)
	baoClient := &fakeAliasClient{aliases: map[string]*openbaoclient.EntityAlias{
		testAliasID: {ID: testAliasID, Name: testAliasName, MountAccessor: testMountAccessor, CanonicalID: testEntityID},
	}}
	reconciler := newAliasReconciler(kubeClient, baoClient)

	if _, err := reconciler.reconcileDeletion(context.Background(), alias); err != nil {
		t.Fatal(err)
	}
	if baoClient.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", baoClient.deleteCalls)
	}
	var got openbaov1alpha1.OpenBaoEntityAlias
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(alias), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Finalizers) != 0 {
		t.Fatalf("finalizers = %v, want none", got.Finalizers)
	}
}

func readyEntity() *openbaov1alpha1.OpenBaoEntity {
	return &openbaov1alpha1.OpenBaoEntity{
		ObjectMeta: metav1.ObjectMeta{Name: testAliasEntityName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoEntitySpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: testConnectionName},
		},
		Status: openbaov1alpha1.OpenBaoEntityStatus{
			ID: testEntityID,
			Conditions: []metav1.Condition{{
				Type: conditionReady, Status: metav1.ConditionTrue, Reason: reasonReconciled,
			}},
		},
	}
}

func newAliasReconciler(kubeClient client.Client, baoClient *fakeAliasClient) *OpenBaoEntityAliasReconciler {
	return &OpenBaoEntityAliasReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (AliasClient, error) {
			return baoClient, nil
		},
	}
}

type fakeAliasClient struct {
	aliases     map[string]*openbaoclient.EntityAlias
	createCalls int
	updateCalls int
	deleteCalls int
}

func (f *fakeAliasClient) ListEntityAliasIDs(context.Context) ([]string, error) {
	ids := make([]string, 0, len(f.aliases))
	for id := range f.aliases {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

func (f *fakeAliasClient) GetEntityAliasByID(_ context.Context, id string) (*openbaoclient.EntityAlias, error) {
	alias, ok := f.aliases[id]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	copy := *alias
	return &copy, nil
}

func (f *fakeAliasClient) CreateEntityAlias(_ context.Context, request openbaoclient.EntityAliasRequest) (string, error) {
	if f.aliases == nil {
		f.aliases = make(map[string]*openbaoclient.EntityAlias)
	}
	f.createCalls++
	f.aliases[testAliasID] = &openbaoclient.EntityAlias{
		ID: testAliasID, Name: request.Name, MountAccessor: request.MountAccessor, CanonicalID: request.CanonicalID,
	}
	return testAliasID, nil
}

func (f *fakeAliasClient) UpdateEntityAlias(_ context.Context, id string, request openbaoclient.EntityAliasRequest) (*openbaoclient.EntityAlias, error) {
	alias, ok := f.aliases[id]
	if !ok {
		return nil, fmt.Errorf("alias %s not found", id)
	}
	f.updateCalls++
	alias.CanonicalID = request.CanonicalID
	copy := *alias
	return &copy, nil
}

func (f *fakeAliasClient) DeleteEntityAlias(_ context.Context, id string) error {
	f.deleteCalls++
	delete(f.aliases, id)
	return nil
}
