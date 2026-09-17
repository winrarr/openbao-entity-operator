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
	testGroupName          = "platform"
	testGroupID            = "group-1"
	testPlatformMembership = "platform-payments"
	testUnmanagedEntityID  = "unmanaged-entity"
	testInternalGroupType  = "internal"
)

func TestGroupReconcilerCreatesGroupAndMembership(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testGroupName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoGroupSpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			Policies:      []string{testDefaultPolicy},
		},
	}
	membership := &openbaov1alpha1.OpenBaoGroupMembership{
		ObjectMeta: metav1.ObjectMeta{Name: testPlatformMembership, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoGroupMembershipSpec{
			GroupRef:  openbaov1alpha1.OpenBaoGroupReference{Name: group.Name},
			EntityRef: &openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
		},
	}
	kubeClient := newTestClient(connection, entity, group, membership)
	baoClient := &fakeGroupClient{}
	reconciler := newGroupReconciler(kubeClient, baoClient)

	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 {
		t.Fatalf("RequeueAfter = %s, want periodic drift check", result.RequeueAfter)
	}

	var gotGroup openbaov1alpha1.OpenBaoGroup
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(group), &gotGroup); err != nil {
		t.Fatal(err)
	}
	if gotGroup.Status.ID != testGroupID || !conditionTrue(gotGroup.Status.Conditions) {
		t.Fatalf("group status = %#v, want group-1 and Ready=True", gotGroup.Status)
	}
	if !slices.Equal(gotGroup.Status.MemberEntityIDs, []string{testEntityID}) || !slices.Equal(gotGroup.Status.ManagedMemberEntityIDs, []string{testEntityID}) {
		t.Fatalf("group members = %v/%v, want entity-1/entity-1", gotGroup.Status.MemberEntityIDs, gotGroup.Status.ManagedMemberEntityIDs)
	}
	var gotMembership openbaov1alpha1.OpenBaoGroupMembership
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(membership), &gotMembership); err != nil {
		t.Fatal(err)
	}
	if gotMembership.Status.GroupID != testGroupID || gotMembership.Status.MemberID != testEntityID || !conditionTrue(gotMembership.Status.Conditions) {
		t.Fatalf("membership status = %#v, want group-1/entity-1 and Ready=True", gotMembership.Status)
	}
	if baoClient.createCalls != 1 || baoClient.updateCalls != 1 {
		t.Fatalf("create/update calls = %d/%d, want 1/1", baoClient.createCalls, baoClient.updateCalls)
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}}); err != nil {
		t.Fatal(err)
	}
	if baoClient.createCalls != 1 || baoClient.updateCalls != 1 {
		t.Fatalf("calls after second reconcile = %d/%d, want 1/1", baoClient.createCalls, baoClient.updateCalls)
	}
}

func TestGroupReconcilerAdoptsExistingGroup(t *testing.T) {
	connection := readyConnection()
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testGroupName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoGroupSpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			CreationPolicy: openbaov1alpha1.CreationPolicyAdopt,
		},
	}
	kubeClient := newTestClient(connection, group)
	baoClient := &fakeGroupClient{groups: map[string]*openbaoclient.Group{
		testGroupID: {ID: testGroupID, Name: testGroupName, Type: testInternalGroupType},
	}}

	if _, err := newGroupReconciler(kubeClient, baoClient).Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}}); err != nil {
		t.Fatal(err)
	}
	var got openbaov1alpha1.OpenBaoGroup
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(group), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.ID != testGroupID || baoClient.createCalls != 0 || !conditionTrue(got.Status.Conditions) {
		t.Fatalf("status/create calls = %#v/%d, want adopted group-1 and no create", got.Status, baoClient.createCalls)
	}
}

func TestGroupReconcilerRefusesUnexpectedAdoption(t *testing.T) {
	connection := readyConnection()
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testGroupName, Namespace: testNamespace},
		Spec:       openbaov1alpha1.OpenBaoGroupSpec{ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name}},
	}
	kubeClient := newTestClient(connection, group)
	baoClient := &fakeGroupClient{groups: map[string]*openbaoclient.Group{
		testGroupID: {ID: testGroupID, Name: testGroupName, Type: testInternalGroupType},
	}}

	if _, err := newGroupReconciler(kubeClient, baoClient).Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}}); err == nil {
		t.Fatal("expected an error when creation policy refuses adoption")
	}
	var got openbaov1alpha1.OpenBaoGroup
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(group), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions)
	if condition == nil || condition.Reason != "GroupAcquireFailed" || condition.Status != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %#v, want GroupAcquireFailed/False", condition)
	}
}

func TestGroupReconcilerRemovesOnlyClaimedMembership(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testGroupName, Namespace: testNamespace},
		Spec:       openbaov1alpha1.OpenBaoGroupSpec{ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name}},
		Status: openbaov1alpha1.OpenBaoGroupStatus{
			ID:                     testGroupID,
			ManagedMemberEntityIDs: []string{testEntityID},
		},
	}
	membership := &openbaov1alpha1.OpenBaoGroupMembership{
		ObjectMeta: metav1.ObjectMeta{Name: testPlatformMembership, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoGroupMembershipSpec{
			GroupRef:  openbaov1alpha1.OpenBaoGroupReference{Name: group.Name},
			EntityRef: &openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
		},
		Status: openbaov1alpha1.OpenBaoGroupMembershipStatus{MemberID: testEntityID, MemberType: "Entity"},
	}
	kubeClient := newTestClient(connection, entity, group, membership)
	baoClient := &fakeGroupClient{groups: map[string]*openbaoclient.Group{
		testGroupID: {ID: testGroupID, Name: testGroupName, Type: testInternalGroupType, MemberEntityIDs: []string{testEntityID, testUnmanagedEntityID}},
	}}
	reconciler := newGroupReconciler(kubeClient, baoClient)
	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(baoClient.groups[testGroupID].MemberEntityIDs, []string{testEntityID, testUnmanagedEntityID}) {
		t.Fatalf("members after claim = %v, want claimed and unmanaged members", baoClient.groups[testGroupID].MemberEntityIDs)
	}
	if err := kubeClient.Delete(context.Background(), membership); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(baoClient.groups[testGroupID].MemberEntityIDs, []string{testUnmanagedEntityID}) {
		t.Fatalf("members after claim deletion = %v, want only unmanaged-entity", baoClient.groups[testGroupID].MemberEntityIDs)
	}
}

func TestGroupReconcilerRejectsMembershipForExternalGroup(t *testing.T) {
	connection := readyConnection()
	entity := readyEntity()
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testGroupName, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoGroupSpec{
			ConnectionRef: openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			Type:          openbaov1alpha1.OpenBaoGroupTypeExternal,
		},
		Status: openbaov1alpha1.OpenBaoGroupStatus{ID: testGroupID},
	}
	membership := &openbaov1alpha1.OpenBaoGroupMembership{
		ObjectMeta: metav1.ObjectMeta{Name: testPlatformMembership, Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoGroupMembershipSpec{
			GroupRef:  openbaov1alpha1.OpenBaoGroupReference{Name: group.Name},
			EntityRef: &openbaov1alpha1.OpenBaoEntityReference{Name: entity.Name},
		},
	}
	kubeClient := newTestClient(connection, entity, group, membership)
	baoClient := &fakeGroupClient{groups: map[string]*openbaoclient.Group{
		testGroupID: {ID: testGroupID, Name: testGroupName, Type: "external"},
	}}
	reconciler := newGroupReconciler(kubeClient, baoClient)

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}}); err == nil {
		t.Fatal("expected external group membership error")
	}
	var got openbaov1alpha1.OpenBaoGroupMembership
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(membership), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions)
	if condition == nil || condition.Reason != "ExternalGroupMembershipUnsupported" {
		t.Fatalf("membership Ready condition = %#v, want ExternalGroupMembershipUnsupported", condition)
	}
}

func TestGroupReconcilerDeletesGroup(t *testing.T) {
	connection := readyConnection()
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testGroupName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoGroupSpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: connection.Name},
			DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
		},
		Status: openbaov1alpha1.OpenBaoGroupStatus{ID: testGroupID},
	}
	kubeClient := newTestClient(connection, group)
	baoClient := &fakeGroupClient{groups: map[string]*openbaoclient.Group{
		testGroupID: {ID: testGroupID, Name: testGroupName, Type: testInternalGroupType},
	}}

	if _, err := newGroupReconciler(kubeClient, baoClient).reconcileDeletion(context.Background(), group); err != nil {
		t.Fatal(err)
	}
	if baoClient.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", baoClient.deleteCalls)
	}
	var got openbaov1alpha1.OpenBaoGroup
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(group), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Finalizers) != 0 {
		t.Fatalf("finalizers = %v, want none", got.Finalizers)
	}
}

func TestGroupReconcilerRetainsFinalizerWhenConnectionIsMissing(t *testing.T) {
	group := &openbaov1alpha1.OpenBaoGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testGroupName,
			Namespace:  testNamespace,
			Finalizers: []string{finalizerName},
		},
		Spec: openbaov1alpha1.OpenBaoGroupSpec{
			ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: testConnectionName},
			DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
		},
		Status: openbaov1alpha1.OpenBaoGroupStatus{ID: testGroupID},
	}
	kubeClient := newTestClient(group)
	reconciler := newGroupReconciler(kubeClient, &fakeGroupClient{})

	result, err := reconciler.reconcileDeletion(context.Background(), group)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter != dependencyRetry {
		t.Fatalf("result = %#v, want retry after %s", result, dependencyRetry)
	}
	var got openbaov1alpha1.OpenBaoGroup
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(group), &got); err != nil {
		t.Fatal(err)
	}
	assertCleanupRequired(t, got.Status.Conditions)
	if len(got.Finalizers) != 1 || got.Finalizers[0] != finalizerName {
		t.Fatalf("finalizers = %v, want %q retained", got.Finalizers, finalizerName)
	}
}

func newGroupReconciler(kubeClient client.Client, baoClient *fakeGroupClient) *OpenBaoGroupReconciler {
	return &OpenBaoGroupReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (GroupClient, error) {
			return baoClient, nil
		},
	}
}

type fakeGroupClient struct {
	groups      map[string]*openbaoclient.Group
	createCalls int
	updateCalls int
	deleteCalls int
}

func (f *fakeGroupClient) GetGroupByID(_ context.Context, id string) (*openbaoclient.Group, error) {
	group, ok := f.groups[id]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	copy := *group
	copy.MemberEntityIDs = append([]string(nil), group.MemberEntityIDs...)
	copy.MemberGroupIDs = append([]string(nil), group.MemberGroupIDs...)
	return &copy, nil
}

func (f *fakeGroupClient) GetGroupByName(_ context.Context, name string) (*openbaoclient.Group, error) {
	for _, group := range f.groups {
		if group.Name == name {
			return f.GetGroupByID(context.Background(), group.ID)
		}
	}
	return nil, &openbaoclient.HTTPError{StatusCode: 404}
}

func (f *fakeGroupClient) CreateGroup(_ context.Context, request openbaoclient.GroupRequest) (string, error) {
	if f.groups == nil {
		f.groups = make(map[string]*openbaoclient.Group)
	}
	f.createCalls++
	f.groups[testGroupID] = &openbaoclient.Group{ID: testGroupID, Name: request.Name, Type: request.Type, Metadata: request.Metadata, Policies: request.Policies, MemberEntityIDs: request.MemberEntityIDs, MemberGroupIDs: request.MemberGroupIDs}
	return testGroupID, nil
}

func (f *fakeGroupClient) UpdateGroup(_ context.Context, id string, request openbaoclient.GroupRequest) (*openbaoclient.Group, error) {
	group, ok := f.groups[id]
	if !ok {
		return nil, fmt.Errorf("group %s not found", id)
	}
	f.updateCalls++
	group.Name = request.Name
	group.Type = request.Type
	group.Metadata = request.Metadata
	group.Policies = request.Policies
	group.MemberEntityIDs = request.MemberEntityIDs
	group.MemberGroupIDs = request.MemberGroupIDs
	return f.GetGroupByID(context.Background(), id)
}

func (f *fakeGroupClient) DeleteGroup(_ context.Context, id string) error {
	f.deleteCalls++
	delete(f.groups, id)
	return nil
}
