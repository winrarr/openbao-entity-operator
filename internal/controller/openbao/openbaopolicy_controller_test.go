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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

func TestPolicyReconcilerCreatesAndPersistsPolicy(t *testing.T) {
	connection := readyConnection()
	policy := &openbaov1alpha1.OpenBaoPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: testPolicyName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoPolicySpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			Rules:         testPolicyRules,
		},
	}
	kubeClient := newTestClient(connection, policy)
	baoClient := &fakePolicyClient{}
	reconciler := &OpenBaoPolicyReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (PolicyClient, error) {
			return baoClient, nil
		},
	}
	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace}}
	if result, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	} else if result.RequeueAfter <= 0 {
		t.Fatalf("RequeueAfter = %s, want periodic drift check", result.RequeueAfter)
	}

	var got openbaov1alpha1.OpenBaoPolicy
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(policy), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Name != testPolicyName || got.Status.Version != 1 || got.Status.RulesHash != policyRulesHash(testPolicyRules) || !conditionTrue(got.Status.Conditions) {
		t.Fatalf("status = %#v, want managed version 1 and Ready=True", got.Status)
	}
	if baoClient.writeCalls != 1 {
		t.Fatalf("write calls = %d, want 1", baoClient.writeCalls)
	}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if baoClient.writeCalls != 1 {
		t.Fatalf("write calls after second reconcile = %d, want 1", baoClient.writeCalls)
	}
}

func TestPolicyReconcilerAdoptsAndCorrectsDrift(t *testing.T) {
	connection := readyConnection()
	policy := &openbaov1alpha1.OpenBaoPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: testPolicyName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoPolicySpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			CreationPolicy: openbaov1alpha1.CreationPolicyAdopt,
			Rules:          testPolicyUpdatedRules,
		},
	}
	kubeClient := newTestClient(connection, policy)
	baoClient := &fakePolicyClient{policies: map[string]*openbaoclient.Policy{
		testPolicyName: {Name: testPolicyName, Rules: testPolicyRules, Version: 3},
	}}
	reconciler := newPolicyReconciler(kubeClient, baoClient)
	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace}}); err != nil {
		t.Fatal(err)
	}
	if baoClient.writeCalls != 1 || baoClient.policies[testPolicyName].Rules != testPolicyUpdatedRules || baoClient.policies[testPolicyName].Version != 4 {
		t.Fatalf("policy = %#v, writes = %d, want updated adopted policy at version 4", baoClient.policies[testPolicyName], baoClient.writeCalls)
	}
}

func TestPolicyReconcilerRefusesUnexpectedAdoption(t *testing.T) {
	connection := readyConnection()
	policy := &openbaov1alpha1.OpenBaoPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: testPolicyName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoPolicySpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			Rules:         testPolicyRules,
		},
	}
	kubeClient := newTestClient(connection, policy)
	baoClient := &fakePolicyClient{policies: map[string]*openbaoclient.Policy{
		testPolicyName: {Name: testPolicyName, Rules: testPolicyRules, Version: 1},
	}}
	reconciler := newPolicyReconciler(kubeClient, baoClient)
	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace}}); err == nil {
		t.Fatal("expected an error when creation policy refuses adoption")
	}
	var got openbaov1alpha1.OpenBaoPolicy
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(policy), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions)
	if condition == nil || condition.Reason != "PolicyAcquireFailed" || condition.Status != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %#v, want PolicyAcquireFailed/False", condition)
	}
	if baoClient.writeCalls != 0 {
		t.Fatalf("write calls = %d, want 0", baoClient.writeCalls)
	}
}

func TestPolicyReconcilerAppliesDeletionPolicy(t *testing.T) {
	connection := readyConnection()
	policy := &openbaov1alpha1.OpenBaoPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testPolicyName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoPolicySpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
			Rules:          testPolicyRules,
		},
	}
	kubeClient := newTestClient(connection, policy)
	baoClient := &fakePolicyClient{policies: map[string]*openbaoclient.Policy{
		testPolicyName: {Name: testPolicyName, Rules: testPolicyRules, Version: 1},
	}}
	reconciler := newPolicyReconciler(kubeClient, baoClient)
	if _, err := reconciler.reconcileDeletion(context.Background(), policy); err != nil {
		t.Fatal(err)
	}
	if baoClient.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", baoClient.deleteCalls)
	}
	if _, ok := baoClient.policies[testPolicyName]; ok {
		t.Fatal("policy still exists after Delete-policy cleanup")
	}
	var got openbaov1alpha1.OpenBaoPolicy
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(policy), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Finalizers) != 0 {
		t.Fatalf("finalizers = %v, want none", got.Finalizers)
	}
}

func TestPolicyReconcilerRetainsFinalizerWhenConnectionIsMissing(t *testing.T) {
	policy := &openbaov1alpha1.OpenBaoPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testPolicyName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoPolicySpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: testConnectionName},
			DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
			Rules:          testPolicyRules,
		},
		Status: openbaov1alpha1.OpenBaoPolicyStatus{Name: testPolicyName},
	}
	kubeClient := newTestClient(policy)
	reconciler := newPolicyReconciler(kubeClient, &fakePolicyClient{})

	result, err := reconciler.reconcileDeletion(context.Background(), policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter != dependencyRetry {
		t.Fatalf("result = %#v, want retry after %s", result, dependencyRetry)
	}
	var got openbaov1alpha1.OpenBaoPolicy
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(policy), &got); err != nil {
		t.Fatal(err)
	}
	assertCleanupRequired(t, got.Status.Conditions)
	if len(got.Finalizers) != 1 || got.Finalizers[0] != finalizerName {
		t.Fatalf("finalizers = %v, want %q retained", got.Finalizers, finalizerName)
	}
}

func newPolicyReconciler(kubeClient client.Client, baoClient *fakePolicyClient) *OpenBaoPolicyReconciler {
	return &OpenBaoPolicyReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (PolicyClient, error) {
			return baoClient, nil
		},
	}
}
