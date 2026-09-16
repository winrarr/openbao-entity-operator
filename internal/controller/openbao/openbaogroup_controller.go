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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

// GroupClient is the OpenBao group surface reconciled by OpenBaoGroup.
type GroupClient interface {
	GetGroupByID(context.Context, string) (*openbaoclient.Group, error)
	GetGroupByName(context.Context, string) (*openbaoclient.Group, error)
	CreateGroup(context.Context, openbaoclient.GroupRequest) (string, error)
	UpdateGroup(context.Context, string, openbaoclient.GroupRequest) (*openbaoclient.Group, error)
	DeleteGroup(context.Context, string) error
}

type groupMembershipClaim struct {
	object     *openbaov1alpha1.OpenBaoGroupMembership
	memberID   string
	memberType string
}

const (
	groupMemberTypeEntity = "Entity"
	groupMemberTypeGroup  = "Group"
)

type membershipResolutionError struct {
	dependency bool
	err        error
}

func (e *membershipResolutionError) Error() string { return e.err.Error() }
func (e *membershipResolutionError) Unwrap() error { return e.err }

// OpenBaoGroupReconciler reconciles OpenBaoGroup resources and their
// OpenBaoGroupMembership claims.
type OpenBaoGroupReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (GroupClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroups/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroupmemberships,verbs=get;list;watch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroupmemberships/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentities,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao group and the membership claims that point to it.
func (r *OpenBaoGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("openbao-group")
	var group openbaov1alpha1.OpenBaoGroup
	if err := r.Get(ctx, req.NamespacedName, &group); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := group.Status.DeepCopy()
	if group.DeletionTimestamp.IsZero() && groupDeletionPolicy(&group).IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &group); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !group.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &group)
	}

	group.Status.ObservedGeneration = group.Generation
	connection, err := resolveConnection(ctx, r.Client, group.Namespace, group.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &group, fmt.Errorf("read OpenBaoConnection %s/%s: %w", group.Namespace, group.Spec.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &group, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}

	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &group, err)
	}

	observed, err := r.acquire(ctx, &group, apiClient)
	if err != nil {
		return r.fail(ctx, &group, "GroupAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &group, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no group for %q", group.Name))
	}
	if observed.Name != "" && observed.Name != group.Name {
		return r.fail(ctx, &group, "GroupIdentityMismatch", fmt.Errorf("OpenBao group %q is named %q, expected %q", group.Status.ID, observed.Name, group.Name))
	}

	memberships, err := r.membershipsForGroup(ctx, &group)
	if err != nil {
		return r.fail(ctx, &group, "GroupMembershipReadFailed", err)
	}
	claims, err := r.resolveMemberships(ctx, &group, memberships)
	if err != nil {
		var resolutionErr *membershipResolutionError
		if errors.As(err, &resolutionErr) && resolutionErr.dependency {
			return r.dependencyFailure(ctx, &group, resolutionErr)
		}
		return r.fail(ctx, &group, "GroupMembershipResolutionFailed", err)
	}

	desired := desiredGroupRequest(&group, observed, claims)
	if !groupMatches(desired, observed) {
		if _, err := apiClient.UpdateGroup(ctx, group.Status.ID, desired); err != nil {
			return r.fail(ctx, &group, "GroupUpdateFailed", err)
		}
		observed, err = apiClient.GetGroupByID(ctx, group.Status.ID)
		if err != nil {
			return r.fail(ctx, &group, "GroupReadFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &group, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no group after updating %q", group.Status.ID))
		}
	}

	observeGroupStatus(&group, observed, managedMemberIDs(claims))
	markReady(&group.Status.Conditions, group.Generation, "OpenBao group is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &group, before, &group.Status); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.updateMembershipStatuses(ctx, &group, claims, observed); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Reconciled OpenBao group", "id", group.Status.ID, "name", group.Name, "entityMembers", len(observed.MemberEntityIDs), "groupMembers", len(observed.MemberGroupIDs))
	return ctrl.Result{RequeueAfter: groupDriftInterval(&group)}, nil
}

func (r *OpenBaoGroupReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (GroupClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	var apiClient *openbaoclient.Client
	var err error
	if r.ClientCache != nil {
		apiClient, err = r.ClientCache.ClientFor(ctx, r.Client, connection)
	} else {
		apiClient, err = connectionClientFor(ctx, r.Client, connection)
	}
	if err != nil {
		return nil, err
	}
	return apiClient, nil
}

func (r *OpenBaoGroupReconciler) acquire(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup, apiClient GroupClient) (*openbaoclient.Group, error) {
	if group.Status.ID != "" {
		observed, err := apiClient.GetGroupByID(ctx, group.Status.ID)
		if err != nil {
			if isNotFound(err) {
				group.Status.ID = ""
				return r.acquire(ctx, group, apiClient)
			}
			return nil, err
		}
		return observed, nil
	}

	observed, err := apiClient.GetGroupByName(ctx, group.Name)
	if err == nil {
		if !groupCreationPolicy(group).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao group %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", group.Name)
		}
		if observed.ID == "" {
			return nil, fmt.Errorf("OpenBao returned a group named %q without an ID", group.Name)
		}
		group.Status.ID = observed.ID
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !groupCreationPolicy(group).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao group %q does not exist and creationPolicy=%s does not allow creation", group.Name, groupCreationPolicy(group))
	}

	id, err := apiClient.CreateGroup(ctx, desiredGroupRequest(group, nil, nil))
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("OpenBao returned an empty ID when creating group %q", group.Name)
	}
	group.Status.ID = id
	return apiClient.GetGroupByID(ctx, id)
}

func (r *OpenBaoGroupReconciler) reconcileDeletion(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup) (ctrl.Result, error) {
	if groupDeletionPolicy(group) != openbaov1alpha1.DeletionPolicyDelete || group.Status.ID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, group)
	}
	connection, err := resolveConnection(ctx, r.Client, group.Namespace, group.Spec.ConnectionRef)
	if err != nil {
		if isNotFound(err) {
			return removeFinalizerAfterDependencyLoss(ctx, r.Client, group, "OpenBaoConnection", err)
		}
		return ctrl.Result{}, err
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		if isNotFound(err) {
			return removeFinalizerAfterDependencyLoss(ctx, r.Client, group, "OpenBaoConnection credentials", err)
		}
		return ctrl.Result{}, err
	}
	if err := apiClient.DeleteGroup(ctx, group.Status.ID); err != nil && !isNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, group)
}

func (r *OpenBaoGroupReconciler) dependencyFailure(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup, err error) (ctrl.Result, error) {
	before := group.Status.DeepCopy()
	group.Status.ObservedGeneration = group.Generation
	markStalled(&group.Status.Conditions, group.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, group, before, &group.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoGroupReconciler) fail(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup, reason string, err error) (ctrl.Result, error) {
	before := group.Status.DeepCopy()
	group.Status.ObservedGeneration = group.Generation
	markError(&group.Status.Conditions, group.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, group, before, &group.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoGroupReconciler) membershipsForGroup(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup) ([]openbaov1alpha1.OpenBaoGroupMembership, error) {
	var memberships openbaov1alpha1.OpenBaoGroupMembershipList
	if err := r.List(ctx, &memberships, client.InNamespace(group.Namespace)); err != nil {
		return nil, err
	}
	result := make([]openbaov1alpha1.OpenBaoGroupMembership, 0, len(memberships.Items))
	for i := range memberships.Items {
		membership := memberships.Items[i]
		if membership.Spec.GroupRef.Name == group.Name {
			result = append(result, membership)
		}
	}
	slices.SortFunc(result, func(a, b openbaov1alpha1.OpenBaoGroupMembership) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

func (r *OpenBaoGroupReconciler) resolveMemberships(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup, memberships []openbaov1alpha1.OpenBaoGroupMembership) ([]groupMembershipClaim, error) {
	if len(memberships) == 0 {
		return nil, nil
	}
	if groupType(group) == openbaov1alpha1.OpenBaoGroupTypeExternal {
		for i := range memberships {
			if err := r.markMembershipError(ctx, &memberships[i], "ExternalGroupMembershipUnsupported", fmt.Errorf("OpenBao external group %q manages membership through its external alias", group.Name)); err != nil {
				return nil, err
			}
		}
		return nil, &membershipResolutionError{err: fmt.Errorf("OpenBao external group %q cannot be managed with OpenBaoGroupMembership resources", group.Name)}
	}

	claims := make([]groupMembershipClaim, 0, len(memberships))
	for i := range memberships {
		membership := &memberships[i]
		if (membership.Spec.EntityRef == nil) == (membership.Spec.MemberGroupRef == nil) {
			err := fmt.Errorf("exactly one of entityRef or memberGroupRef must be set")
			if statusErr := r.markMembershipError(ctx, membership, "InvalidMembership", err); statusErr != nil {
				return nil, statusErr
			}
			return nil, &membershipResolutionError{err: err}
		}

		if membership.Spec.EntityRef != nil {
			entity, err := resolveEntity(ctx, r.Client, group.Namespace, *membership.Spec.EntityRef)
			if err != nil {
				dependencyErr := dependencyMessage("OpenBaoEntity", types.NamespacedName{Namespace: group.Namespace, Name: membership.Spec.EntityRef.Name}, err)
				if statusErr := r.markMembershipStalled(ctx, membership, dependencyErr); statusErr != nil {
					return nil, statusErr
				}
				return nil, &membershipResolutionError{dependency: true, err: err}
			}
			if !meta.IsStatusConditionTrue(entity.Status.Conditions, conditionReady) || entity.Status.ID == "" {
				err := dependencyMessage("OpenBaoEntity", client.ObjectKeyFromObject(entity), nil)
				if statusErr := r.markMembershipStalled(ctx, membership, err); statusErr != nil {
					return nil, statusErr
				}
				return nil, &membershipResolutionError{dependency: true, err: err}
			}
			claims = append(claims, groupMembershipClaim{object: membership, memberID: entity.Status.ID, memberType: groupMemberTypeEntity})
			continue
		}

		memberGroup := membership.Spec.MemberGroupRef
		if memberGroup.Name == group.Name {
			err := fmt.Errorf("OpenBaoGroupMembership %s/%s cannot make group %q a member of itself", membership.Namespace, membership.Name, group.Name)
			if statusErr := r.markMembershipError(ctx, membership, "InvalidMembership", err); statusErr != nil {
				return nil, statusErr
			}
			return nil, &membershipResolutionError{err: err}
		}
		child, err := resolveGroup(ctx, r.Client, group.Namespace, *memberGroup)
		if err != nil {
			dependencyErr := dependencyMessage("OpenBaoGroup", types.NamespacedName{Namespace: group.Namespace, Name: memberGroup.Name}, err)
			if statusErr := r.markMembershipStalled(ctx, membership, dependencyErr); statusErr != nil {
				return nil, statusErr
			}
			return nil, &membershipResolutionError{dependency: true, err: err}
		}
		if !meta.IsStatusConditionTrue(child.Status.Conditions, conditionReady) || child.Status.ID == "" {
			err := dependencyMessage("OpenBaoGroup", client.ObjectKeyFromObject(child), nil)
			if statusErr := r.markMembershipStalled(ctx, membership, err); statusErr != nil {
				return nil, statusErr
			}
			return nil, &membershipResolutionError{dependency: true, err: err}
		}
		claims = append(claims, groupMembershipClaim{object: membership, memberID: child.Status.ID, memberType: groupMemberTypeGroup})
	}
	return claims, nil
}

func (r *OpenBaoGroupReconciler) updateMembershipStatuses(ctx context.Context, group *openbaov1alpha1.OpenBaoGroup, claims []groupMembershipClaim, observed *openbaoclient.Group) error {
	for _, claim := range claims {
		membership := claim.object
		before := membership.Status.DeepCopy()
		membership.Status.GroupID = group.Status.ID
		membership.Status.MemberID = claim.memberID
		membership.Status.MemberType = claim.memberType
		membership.Status.ObservedGeneration = membership.Generation
		if !containsMember(observed, claim.memberType, claim.memberID) {
			markError(&membership.Status.Conditions, membership.Generation, "MembershipNotApplied", fmt.Errorf("OpenBao group %q does not contain %s %q", group.Name, strings.ToLower(claim.memberType), claim.memberID))
		} else {
			markReady(&membership.Status.Conditions, membership.Generation, "OpenBao group membership is reconciled")
		}
		if err := updateStatusIfChanged(ctx, r.Client, membership, before, &membership.Status); err != nil {
			return err
		}
	}
	return nil
}

func (r *OpenBaoGroupReconciler) markMembershipError(ctx context.Context, membership *openbaov1alpha1.OpenBaoGroupMembership, reason string, err error) error {
	before := membership.Status.DeepCopy()
	membership.Status.ObservedGeneration = membership.Generation
	markError(&membership.Status.Conditions, membership.Generation, reason, err)
	return updateStatusIfChanged(ctx, r.Client, membership, before, &membership.Status)
}

func (r *OpenBaoGroupReconciler) markMembershipStalled(ctx context.Context, membership *openbaov1alpha1.OpenBaoGroupMembership, err error) error {
	before := membership.Status.DeepCopy()
	membership.Status.ObservedGeneration = membership.Generation
	markStalled(&membership.Status.Conditions, membership.Generation, "DependencyNotReady", err)
	return updateStatusIfChanged(ctx, r.Client, membership, before, &membership.Status)
}

func (r *OpenBaoGroupReconciler) mapConnectionToGroups(ctx context.Context, obj client.Object) []reconcile.Request {
	var groups openbaov1alpha1.OpenBaoGroupList
	if err := r.List(ctx, &groups, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range groups.Items {
		group := &groups.Items[i]
		if group.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(group)})
		}
	}
	return requests
}

func (r *OpenBaoGroupReconciler) mapMembershipToGroup(_ context.Context, obj client.Object) []reconcile.Request {
	membership, ok := obj.(*openbaov1alpha1.OpenBaoGroupMembership)
	if !ok || membership.Spec.GroupRef.Name == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: membership.Namespace, Name: membership.Spec.GroupRef.Name}}}
}

func (r *OpenBaoGroupReconciler) mapEntityToGroups(ctx context.Context, obj client.Object) []reconcile.Request {
	var memberships openbaov1alpha1.OpenBaoGroupMembershipList
	if err := r.List(ctx, &memberships, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range memberships.Items {
		membership := &memberships.Items[i]
		if membership.Spec.EntityRef != nil && membership.Spec.EntityRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: membership.Namespace, Name: membership.Spec.GroupRef.Name}})
		}
	}
	return requests
}

func (r *OpenBaoGroupReconciler) mapGroupToParentGroups(ctx context.Context, obj client.Object) []reconcile.Request {
	var memberships openbaov1alpha1.OpenBaoGroupMembershipList
	if err := r.List(ctx, &memberships, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range memberships.Items {
		membership := &memberships.Items[i]
		if membership.Spec.MemberGroupRef != nil && membership.Spec.MemberGroupRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: membership.Namespace, Name: membership.Spec.GroupRef.Name}})
		}
	}
	return requests
}

// SetupWithManager sets up the group controller and its dependency watches.
func (r *OpenBaoGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoGroup{}).
		Named("openbao-openbaogroup").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToGroups)).
		Watches(&openbaov1alpha1.OpenBaoGroupMembership{}, handler.EnqueueRequestsFromMapFunc(r.mapMembershipToGroup)).
		Watches(&openbaov1alpha1.OpenBaoEntity{}, handler.EnqueueRequestsFromMapFunc(r.mapEntityToGroups)).
		Watches(&openbaov1alpha1.OpenBaoGroup{}, handler.EnqueueRequestsFromMapFunc(r.mapGroupToParentGroups)).
		Complete(r)
}

func desiredGroupRequest(group *openbaov1alpha1.OpenBaoGroup, observed *openbaoclient.Group, claims []groupMembershipClaim) openbaoclient.GroupRequest {
	previousManaged := openbaoclient.Group{
		MemberEntityIDs: group.Status.ManagedMemberEntityIDs,
		MemberGroupIDs:  group.Status.ManagedMemberGroupIDs,
	}
	entityIDs := subtractIDs(observedMemberIDs(observed, "Entity"), previousManaged.MemberEntityIDs)
	groupIDs := subtractIDs(observedMemberIDs(observed, "Group"), previousManaged.MemberGroupIDs)
	for _, claim := range claims {
		if claim.memberType == groupMemberTypeEntity {
			entityIDs = append(entityIDs, claim.memberID)
		} else {
			groupIDs = append(groupIDs, claim.memberID)
		}
	}
	return openbaoclient.GroupRequest{
		Name:            group.Name,
		Type:            strings.ToLower(string(groupType(group))),
		Metadata:        maps.Clone(group.Spec.Metadata),
		Policies:        normalizedIDs(group.Spec.Policies),
		MemberEntityIDs: normalizedIDs(entityIDs),
		MemberGroupIDs:  normalizedIDs(groupIDs),
	}
}

func groupMatches(desired openbaoclient.GroupRequest, current *openbaoclient.Group) bool {
	return current.Name == desired.Name && strings.EqualFold(current.Type, desired.Type) &&
		maps.Equal(current.Metadata, desired.Metadata) &&
		slices.Equal(normalizedIDs(current.Policies), desired.Policies) &&
		slices.Equal(normalizedIDs(current.MemberEntityIDs), desired.MemberEntityIDs) &&
		slices.Equal(normalizedIDs(current.MemberGroupIDs), desired.MemberGroupIDs)
}

func observeGroupStatus(group *openbaov1alpha1.OpenBaoGroup, observed *openbaoclient.Group, managed [2][]string) {
	group.Status.ID = observed.ID
	group.Status.Name = observed.Name
	group.Status.Type = openbaov1alpha1.OpenBaoGroupType(titleCaseGroupType(observed.Type))
	group.Status.Metadata = copyStringMap(observed.Metadata)
	group.Status.Policies = normalizedIDs(observed.Policies)
	group.Status.MemberEntityIDs = normalizedIDs(observed.MemberEntityIDs)
	group.Status.MemberGroupIDs = normalizedIDs(observed.MemberGroupIDs)
	group.Status.ManagedMemberEntityIDs = normalizedIDs(managed[0])
	group.Status.ManagedMemberGroupIDs = normalizedIDs(managed[1])
	group.Status.ObservedGeneration = group.Generation
}

func managedMemberIDs(claims []groupMembershipClaim) [2][]string {
	var managed [2][]string
	for _, claim := range claims {
		if claim.memberType == groupMemberTypeEntity {
			managed[0] = append(managed[0], claim.memberID)
		} else {
			managed[1] = append(managed[1], claim.memberID)
		}
	}
	return managed
}

func containsMember(group *openbaoclient.Group, memberType, memberID string) bool {
	return slices.Contains(observedMemberIDs(group, memberType), memberID)
}

func observedMemberIDs(group *openbaoclient.Group, memberType string) []string {
	if group == nil {
		return nil
	}
	if memberType == groupMemberTypeEntity {
		return group.MemberEntityIDs
	}
	return group.MemberGroupIDs
}

func subtractIDs(source, values []string) []string {
	result := make([]string, 0, len(source))
	for _, candidate := range source {
		if !slices.Contains(values, candidate) {
			result = append(result, candidate)
		}
	}
	return result
}

func normalizedIDs(values []string) []string {
	result := append([]string{}, values...)
	slices.Sort(result)
	return slices.Compact(result)
}

func titleCaseGroupType(value string) string {
	if value == "" {
		return string(openbaov1alpha1.OpenBaoGroupTypeInternal)
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func groupType(group *openbaov1alpha1.OpenBaoGroup) openbaov1alpha1.OpenBaoGroupType {
	if group.Spec.Type == "" {
		return openbaov1alpha1.OpenBaoGroupTypeInternal
	}
	return group.Spec.Type
}

func groupCreationPolicy(group *openbaov1alpha1.OpenBaoGroup) openbaov1alpha1.CreationPolicy {
	if group.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return group.Spec.CreationPolicy
}

func groupDeletionPolicy(group *openbaov1alpha1.OpenBaoGroup) openbaov1alpha1.DeletionPolicy {
	if group.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return group.Spec.DeletionPolicy
}

func groupDriftInterval(group *openbaov1alpha1.OpenBaoGroup) time.Duration {
	if group.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return group.Spec.DriftDetectionInterval.Duration
}
