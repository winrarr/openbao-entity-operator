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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

// GroupAliasClient is the OpenBao identity group-alias surface reconciled by OpenBaoGroupAlias.
type GroupAliasClient interface {
	ListGroupAliasIDs(context.Context) ([]string, error)
	GetGroupAliasByID(context.Context, string) (*openbaoclient.GroupAlias, error)
	CreateGroupAlias(context.Context, openbaoclient.GroupAliasRequest) (string, error)
	UpdateGroupAlias(context.Context, string, openbaoclient.GroupAliasRequest) (*openbaoclient.GroupAlias, error)
	DeleteGroupAlias(context.Context, string) error
}

// OpenBaoGroupAliasReconciler reconciles OpenBaoGroupAlias resources.
type OpenBaoGroupAliasReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (GroupAliasClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroupaliases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroupaliases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroupaliases/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaogroups,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles the OpenBao group alias identified by req.
func (r *OpenBaoGroupAliasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("openbao-group-alias")
	var alias openbaov1alpha1.OpenBaoGroupAlias
	if err := r.Get(ctx, req.NamespacedName, &alias); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := alias.Status.DeepCopy()
	if alias.DeletionTimestamp.IsZero() && groupAliasDeletionPolicy(&alias).IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &alias); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !alias.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &alias)
	}

	alias.Status.ObservedGeneration = alias.Generation
	connection, err := resolveConnection(ctx, r.Client, alias.Namespace, alias.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &alias, fmt.Errorf("read OpenBaoConnection %s/%s: %w", alias.Namespace, alias.Spec.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &alias, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}

	group, err := resolveGroup(ctx, r.Client, alias.Namespace, alias.Spec.GroupRef)
	if err != nil {
		return r.dependencyFailure(ctx, &alias, fmt.Errorf("read OpenBaoGroup %s/%s: %w", alias.Namespace, alias.Spec.GroupRef.Name, err))
	}
	if group.Spec.Type != openbaov1alpha1.OpenBaoGroupTypeExternal {
		return r.fail(ctx, &alias, "InvalidGroupType", fmt.Errorf("OpenBaoGroup %s/%s must have type External", group.Namespace, group.Name))
	}
	if !meta.IsStatusConditionTrue(group.Status.Conditions, conditionReady) || group.Status.ID == "" {
		return r.dependencyFailure(ctx, &alias, dependencyMessage("OpenBaoGroup", client.ObjectKeyFromObject(group), nil))
	}

	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &alias, err)
	}

	observed, err := r.acquire(ctx, &alias, group.Status.ID, apiClient)
	if err != nil {
		return r.fail(ctx, &alias, "GroupAliasAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &alias, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no group alias for %q", alias.Spec.Name))
	}
	if observed.Name != "" && observed.Name != alias.Spec.Name {
		return r.fail(ctx, &alias, "GroupAliasIdentityMismatch", fmt.Errorf("OpenBao group alias %q is named %q, expected %q", alias.Status.ID, observed.Name, alias.Spec.Name))
	}
	if observed.MountAccessor != "" && observed.MountAccessor != alias.Spec.MountAccessor {
		return r.fail(ctx, &alias, "GroupAliasIdentityMismatch", fmt.Errorf("OpenBao group alias %q uses mount accessor %q, expected %q", alias.Status.ID, observed.MountAccessor, alias.Spec.MountAccessor))
	}

	desired := desiredGroupAliasRequest(&alias, group.Status.ID)
	if !groupAliasMatches(desired, observed) {
		observed, err = apiClient.UpdateGroupAlias(ctx, alias.Status.ID, desired)
		if err != nil {
			return r.fail(ctx, &alias, "GroupAliasUpdateFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &alias, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no group alias after updating %q", alias.Status.ID))
		}
		observed, err = apiClient.GetGroupAliasByID(ctx, alias.Status.ID)
		if err != nil {
			return r.fail(ctx, &alias, "GroupAliasReadFailed", err)
		}
	}

	observeGroupAliasStatus(&alias, observed)
	markReady(&alias.Status.Conditions, alias.Generation, "OpenBao group alias is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &alias, before, &alias.Status); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Reconciled OpenBao group alias", "id", alias.Status.ID, "name", alias.Spec.Name, "groupID", alias.Status.CanonicalID)
	return ctrl.Result{RequeueAfter: groupAliasDriftInterval(&alias)}, nil
}

func (r *OpenBaoGroupAliasReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (GroupAliasClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoGroupAliasReconciler) acquire(ctx context.Context, alias *openbaov1alpha1.OpenBaoGroupAlias, groupID string, apiClient GroupAliasClient) (*openbaoclient.GroupAlias, error) {
	if alias.Status.ID != "" {
		observed, err := apiClient.GetGroupAliasByID(ctx, alias.Status.ID)
		if err != nil {
			if isNotFound(err) {
				alias.Status.ID = ""
				return r.acquire(ctx, alias, groupID, apiClient)
			}
			return nil, err
		}
		return observed, nil
	}

	ids, err := apiClient.ListGroupAliasIDs(ctx)
	if err != nil && !isNotFound(err) {
		return nil, err
	}
	if err == nil {
		for _, id := range ids {
			if id == "" {
				continue
			}
			candidate, readErr := apiClient.GetGroupAliasByID(ctx, id)
			if readErr != nil {
				if isNotFound(readErr) {
					continue
				}
				return nil, readErr
			}
			if candidate == nil || candidate.Name != alias.Spec.Name || candidate.MountAccessor != alias.Spec.MountAccessor {
				continue
			}
			if !groupAliasCreationPolicy(alias).AllowsAdoption() {
				return nil, fmt.Errorf("OpenBao group alias %q already exists for mount accessor %q; set creationPolicy to Adopt or CreateOrAdopt to manage it", alias.Spec.Name, alias.Spec.MountAccessor)
			}
			alias.Status.ID = candidate.ID
			return candidate, nil
		}
	}

	if !groupAliasCreationPolicy(alias).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao group alias %q does not exist and creationPolicy=%s does not allow creation", alias.Spec.Name, groupAliasCreationPolicy(alias))
	}
	id, err := apiClient.CreateGroupAlias(ctx, desiredGroupAliasRequest(alias, groupID))
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("OpenBao returned an empty ID when creating group alias %q", alias.Spec.Name)
	}
	alias.Status.ID = id
	return apiClient.GetGroupAliasByID(ctx, id)
}

func (r *OpenBaoGroupAliasReconciler) reconcileDeletion(ctx context.Context, alias *openbaov1alpha1.OpenBaoGroupAlias) (ctrl.Result, error) {
	if groupAliasDeletionPolicy(alias) != openbaov1alpha1.DeletionPolicyDelete || alias.Status.ID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, alias)
	}
	before := alias.Status.DeepCopy()
	alias.Status.ObservedGeneration = alias.Generation
	connection, err := resolveConnection(ctx, r.Client, alias.Namespace, alias.Spec.ConnectionRef)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, alias, before, &alias.Status, &alias.Status.Conditions, alias.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, alias, "OpenBaoConnection", err))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, alias, before, &alias.Status, &alias.Status.Conditions, alias.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, alias, "OpenBaoConnection credentials", err))
	}
	if err := apiClient.DeleteGroupAlias(ctx, alias.Status.ID); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, alias, before, &alias.Status, &alias.Status.Conditions, alias.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, alias)
}

func (r *OpenBaoGroupAliasReconciler) dependencyFailure(ctx context.Context, alias *openbaov1alpha1.OpenBaoGroupAlias, err error) (ctrl.Result, error) {
	before := alias.Status.DeepCopy()
	alias.Status.ObservedGeneration = alias.Generation
	markStalled(&alias.Status.Conditions, alias.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, alias, before, &alias.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoGroupAliasReconciler) fail(ctx context.Context, alias *openbaov1alpha1.OpenBaoGroupAlias, reason string, err error) (ctrl.Result, error) {
	before := alias.Status.DeepCopy()
	alias.Status.ObservedGeneration = alias.Generation
	markError(&alias.Status.Conditions, alias.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, alias, before, &alias.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoGroupAliasReconciler) mapConnectionToAliases(ctx context.Context, obj client.Object) []reconcile.Request {
	var aliases openbaov1alpha1.OpenBaoGroupAliasList
	if err := r.List(ctx, &aliases, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range aliases.Items {
		alias := &aliases.Items[i]
		if alias.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(alias)})
		}
	}
	return requests
}

func (r *OpenBaoGroupAliasReconciler) mapGroupToAliases(ctx context.Context, obj client.Object) []reconcile.Request {
	var aliases openbaov1alpha1.OpenBaoGroupAliasList
	if err := r.List(ctx, &aliases, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range aliases.Items {
		alias := &aliases.Items[i]
		if alias.Spec.GroupRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(alias)})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenBaoGroupAliasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoGroupAlias{}).
		Named("openbao-openbaogroupalias").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToAliases)).
		Watches(&openbaov1alpha1.OpenBaoGroup{}, handler.EnqueueRequestsFromMapFunc(r.mapGroupToAliases)).
		Complete(r)
}

func desiredGroupAliasRequest(alias *openbaov1alpha1.OpenBaoGroupAlias, groupID string) openbaoclient.GroupAliasRequest {
	return openbaoclient.GroupAliasRequest{Name: alias.Spec.Name, MountAccessor: alias.Spec.MountAccessor, CanonicalID: groupID}
}

func groupAliasMatches(desired openbaoclient.GroupAliasRequest, current *openbaoclient.GroupAlias) bool {
	return current.Name == desired.Name && current.MountAccessor == desired.MountAccessor && current.CanonicalID == desired.CanonicalID
}

func observeGroupAliasStatus(alias *openbaov1alpha1.OpenBaoGroupAlias, observed *openbaoclient.GroupAlias) {
	alias.Status.ID = observed.ID
	alias.Status.CanonicalID = observed.CanonicalID
	alias.Status.Name = observed.Name
	alias.Status.MountAccessor = observed.MountAccessor
	alias.Status.ObservedGeneration = alias.Generation
}

func groupAliasDriftInterval(alias *openbaov1alpha1.OpenBaoGroupAlias) time.Duration {
	if alias.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return alias.Spec.DriftDetectionInterval.Duration
}

func groupAliasCreationPolicy(alias *openbaov1alpha1.OpenBaoGroupAlias) openbaov1alpha1.CreationPolicy {
	if alias.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return alias.Spec.CreationPolicy
}

func groupAliasDeletionPolicy(alias *openbaov1alpha1.OpenBaoGroupAlias) openbaov1alpha1.DeletionPolicy {
	if alias.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return alias.Spec.DeletionPolicy
}
