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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const testKubernetesAuthRoleName = "payments-workload"

func TestKubernetesAuthRoleReconcilerCreatesAndPersistsRole(t *testing.T) {
	connection := readyConnection()
	role := &openbaov1alpha1.OpenBaoKubernetesAuthRole{
		ObjectMeta: metav1.ObjectMeta{Name: testKubernetesAuthRoleName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoKubernetesAuthRoleSpec{
			ConnectionRef:                 openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			BoundServiceAccountNames:      []string{testEntityName, "payments-reader"},
			BoundServiceAccountNamespaces: []string{testNamespace},
			TokenPolicies:                 []string{testEntityName, testDefaultPolicy},
			TokenPeriod:                   &metav1.Duration{Duration: 5 * time.Minute},
		},
	}
	kubeClient := newTestClient(connection, role)
	baoClient := &fakeKubernetesAuthRoleClient{}
	reconciler := newKubernetesAuthRoleReconciler(kubeClient, baoClient)
	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: role.Name, Namespace: role.Namespace}}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 {
		t.Fatalf("RequeueAfter = %s, want periodic drift check", result.RequeueAfter)
	}
	var got openbaov1alpha1.OpenBaoKubernetesAuthRole
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(role), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Name != role.Name || got.Status.MountPath != defaultKubernetesAuthRoleMount || got.Status.ConfigHash == "" || !conditionTrue(got.Status.Conditions) {
		t.Fatalf("status = %#v, want observed role and Ready=True", got.Status)
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

func TestKubernetesAuthRoleReconcilerAdoptsAndCorrectsDrift(t *testing.T) {
	connection := readyConnection()
	role := &openbaov1alpha1.OpenBaoKubernetesAuthRole{
		ObjectMeta: metav1.ObjectMeta{Name: testKubernetesAuthRoleName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoKubernetesAuthRoleSpec{
			ConnectionRef:                 openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			CreationPolicy:                openbaov1alpha1.CreationPolicyAdopt,
			BoundServiceAccountNames:      []string{testEntityName},
			BoundServiceAccountNamespaces: []string{testNamespace},
			TokenPolicies:                 []string{testEntityName},
		},
	}
	kubeClient := newTestClient(connection, role)
	baoClient := &fakeKubernetesAuthRoleClient{roles: map[string]*openbaoclient.KubernetesAuthRole{
		role.Name: {
			BoundServiceAccountNames:      []string{"old-workload"},
			BoundServiceAccountNamespaces: []string{testNamespace},
			TokenPolicies:                 []string{"old-policy"},
		},
	}}
	reconciler := newKubernetesAuthRoleReconciler(kubeClient, baoClient)

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(role)}); err != nil {
		t.Fatal(err)
	}
	observed := baoClient.roles[role.Name]
	if baoClient.writeCalls != 1 || len(observed.BoundServiceAccountNames) != 1 || observed.BoundServiceAccountNames[0] != testEntityName || observed.TokenPolicies[0] != testEntityName {
		t.Fatalf("role = %#v, writes = %d, want adopted role corrected once", observed, baoClient.writeCalls)
	}
}

func TestKubernetesAuthRoleReconcilerRefusesUnexpectedAdoption(t *testing.T) {
	connection := readyConnection()
	role := &openbaov1alpha1.OpenBaoKubernetesAuthRole{
		ObjectMeta: metav1.ObjectMeta{Name: testKubernetesAuthRoleName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoKubernetesAuthRoleSpec{
			ConnectionRef:                 openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			BoundServiceAccountNames:      []string{testEntityName},
			BoundServiceAccountNamespaces: []string{testNamespace},
		},
	}
	kubeClient := newTestClient(connection, role)
	baoClient := &fakeKubernetesAuthRoleClient{roles: map[string]*openbaoclient.KubernetesAuthRole{
		role.Name: {BoundServiceAccountNames: []string{testEntityName}, BoundServiceAccountNamespaces: []string{testNamespace}},
	}}
	reconciler := newKubernetesAuthRoleReconciler(kubeClient, baoClient)

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(role)}); err == nil {
		t.Fatal("expected an error when creation policy refuses adoption")
	}
	var got openbaov1alpha1.OpenBaoKubernetesAuthRole
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(role), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions)
	if condition == nil || condition.Reason != "KubernetesAuthRoleAcquireFailed" || condition.Status != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %#v, want KubernetesAuthRoleAcquireFailed/False", condition)
	}
	if baoClient.writeCalls != 0 {
		t.Fatalf("write calls = %d, want 0", baoClient.writeCalls)
	}
}

func TestKubernetesAuthRoleReconcilerAppliesDeletionPolicy(t *testing.T) {
	connection := readyConnection()
	role := &openbaov1alpha1.OpenBaoKubernetesAuthRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testKubernetesAuthRoleName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoKubernetesAuthRoleSpec{
			ConnectionRef:                 openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			DeletionPolicy:                openbaov1alpha1.DeletionPolicyDelete,
			BoundServiceAccountNames:      []string{testEntityName},
			BoundServiceAccountNamespaces: []string{testNamespace},
		},
	}
	kubeClient := newTestClient(connection, role)
	baoClient := &fakeKubernetesAuthRoleClient{roles: map[string]*openbaoclient.KubernetesAuthRole{role.Name: {}}}
	reconciler := newKubernetesAuthRoleReconciler(kubeClient, baoClient)

	if _, err := reconciler.reconcileDeletion(context.Background(), role); err != nil {
		t.Fatal(err)
	}
	if baoClient.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", baoClient.deleteCalls)
	}
	if _, ok := baoClient.roles[role.Name]; ok {
		t.Fatal("role still exists after Delete-policy cleanup")
	}
	var got openbaov1alpha1.OpenBaoKubernetesAuthRole
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(role), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Finalizers) != 0 {
		t.Fatalf("finalizers = %v, want none", got.Finalizers)
	}
}

func TestKubernetesAuthRoleReconcilerRetainsFinalizerWhenConnectionIsMissing(t *testing.T) {
	role := &openbaov1alpha1.OpenBaoKubernetesAuthRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testKubernetesAuthRoleName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoKubernetesAuthRoleSpec{
			ConnectionRef:                 openbaov1alpha1.OpenBaoConnectionReference{Name: testConnectionName},
			DeletionPolicy:                openbaov1alpha1.DeletionPolicyDelete,
			BoundServiceAccountNames:      []string{"payments"},
			BoundServiceAccountNamespaces: []string{testNamespace},
		},
		Status: openbaov1alpha1.OpenBaoKubernetesAuthRoleStatus{Name: testKubernetesAuthRoleName},
	}
	kubeClient := newTestClient(role)
	reconciler := &OpenBaoKubernetesAuthRoleReconciler{Client: kubeClient}

	result, err := reconciler.reconcileDeletion(context.Background(), role)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter != dependencyRetry {
		t.Fatalf("result = %#v, want retry after %s", result, dependencyRetry)
	}
	var got openbaov1alpha1.OpenBaoKubernetesAuthRole
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(role), &got); err != nil {
		t.Fatal(err)
	}
	assertCleanupRequired(t, got.Status.Conditions)
	if len(got.Finalizers) != 1 || got.Finalizers[0] != finalizerName {
		t.Fatalf("finalizers = %v, want %q retained", got.Finalizers, finalizerName)
	}
}

func newKubernetesAuthRoleReconciler(kubeClient client.Client, baoClient *fakeKubernetesAuthRoleClient) *OpenBaoKubernetesAuthRoleReconciler {
	return &OpenBaoKubernetesAuthRoleReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (KubernetesAuthRoleClient, error) {
			return baoClient, nil
		},
	}
}

type fakeKubernetesAuthRoleClient struct {
	roles       map[string]*openbaoclient.KubernetesAuthRole
	writeCalls  int
	deleteCalls int
}

func (f *fakeKubernetesAuthRoleClient) GetKubernetesAuthRole(_ context.Context, _, name string) (*openbaoclient.KubernetesAuthRole, error) {
	role, ok := f.roles[name]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	copy := *role
	copy.BoundServiceAccountNames = append([]string(nil), role.BoundServiceAccountNames...)
	copy.BoundServiceAccountNamespaces = append([]string(nil), role.BoundServiceAccountNamespaces...)
	copy.TokenPolicies = append([]string(nil), role.TokenPolicies...)
	copy.Policies = append([]string(nil), role.Policies...)
	return &copy, nil
}

func (f *fakeKubernetesAuthRoleClient) WriteKubernetesAuthRole(_ context.Context, _, name string, request openbaoclient.KubernetesAuthRoleRequest) error {
	if f.roles == nil {
		f.roles = make(map[string]*openbaoclient.KubernetesAuthRole)
	}
	f.writeCalls++
	f.roles[name] = &openbaoclient.KubernetesAuthRole{
		BoundServiceAccountNames:      append([]string(nil), request.BoundServiceAccountNames...),
		BoundServiceAccountNamespaces: append([]string(nil), request.BoundServiceAccountNamespaces...),
		TokenPolicies:                 append([]string(nil), request.TokenPolicies...),
		TokenTTL:                      request.TokenTTL,
		TokenMaxTTL:                   request.TokenMaxTTL,
		TokenPeriod:                   request.TokenPeriod,
	}
	return nil
}

func (f *fakeKubernetesAuthRoleClient) DeleteKubernetesAuthRole(_ context.Context, _, name string) error {
	f.deleteCalls++
	if _, ok := f.roles[name]; !ok {
		return &openbaoclient.HTTPError{StatusCode: 404}
	}
	delete(f.roles, name)
	return nil
}
