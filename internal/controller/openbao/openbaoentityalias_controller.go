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

// AliasClient is the OpenBao entity-alias surface reconciled by OpenBaoEntityAlias.
type AliasClient interface {
	ListEntityAliasIDs(context.Context) ([]string, error)
	GetEntityAliasByID(context.Context, string) (*openbaoclient.EntityAlias, error)
	CreateEntityAlias(context.Context, openbaoclient.EntityAliasRequest) (string, error)
	UpdateEntityAlias(context.Context, string, openbaoclient.EntityAliasRequest) (*openbaoclient.EntityAlias, error)
	DeleteEntityAlias(context.Context, string) error
}

// OpenBaoEntityAliasReconciler reconciles an OpenBaoEntityAlias object.
type OpenBaoEntityAliasReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	NewClient func(context.Context, *openbaov1alpha1.OpenBaoConnection) (AliasClient, error)
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentityaliases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentityaliases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentityaliases/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentities,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles the OpenBao alias identified by req.
func (r *OpenBaoEntityAliasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("openbao-entity-alias")
	var alias openbaov1alpha1.OpenBaoEntityAlias
	if err := r.Get(ctx, req.NamespacedName, &alias); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := alias.Status.DeepCopy()
	if alias.DeletionTimestamp.IsZero() && aliasDeletionPolicy(&alias).IsDelete() {
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

	entity, err := resolveEntity(ctx, r.Client, alias.Namespace, alias.Spec.EntityRef)
	if err != nil {
		return r.dependencyFailure(ctx, &alias, fmt.Errorf("read OpenBaoEntity %s/%s: %w", alias.Namespace, alias.Spec.EntityRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(entity.Status.Conditions, conditionReady) || entity.Status.ID == "" {
		return r.dependencyFailure(ctx, &alias, dependencyMessage("OpenBaoEntity", client.ObjectKeyFromObject(entity), nil))
	}

	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &alias, err)
	}

	observed, err := r.acquire(ctx, &alias, entity.Status.ID, apiClient)
	if err != nil {
		return r.fail(ctx, &alias, "AliasAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &alias, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no alias for %q", alias.Spec.Name))
	}
	if observed.Name != "" && observed.Name != alias.Spec.Name {
		return r.fail(ctx, &alias, "AliasIdentityMismatch", fmt.Errorf("OpenBao alias %q is named %q, expected %q", alias.Status.ID, observed.Name, alias.Spec.Name))
	}
	if observed.MountAccessor != "" && observed.MountAccessor != alias.Spec.MountAccessor {
		return r.fail(ctx, &alias, "AliasIdentityMismatch", fmt.Errorf("OpenBao alias %q uses mount accessor %q, expected %q", alias.Status.ID, observed.MountAccessor, alias.Spec.MountAccessor))
	}

	desired := desiredEntityAliasRequest(&alias, entity.Status.ID)
	if !entityAliasMatches(desired, observed) {
		observed, err = apiClient.UpdateEntityAlias(ctx, alias.Status.ID, openbaoclient.EntityAliasRequest{CanonicalID: desired.CanonicalID})
		if err != nil {
			return r.fail(ctx, &alias, "AliasUpdateFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &alias, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no alias after updating %q", alias.Status.ID))
		}
		observed, err = apiClient.GetEntityAliasByID(ctx, alias.Status.ID)
		if err != nil {
			return r.fail(ctx, &alias, "AliasReadFailed", err)
		}
	}

	if observed.Name != "" && observed.Name != alias.Spec.Name {
		return r.fail(ctx, &alias, "AliasIdentityMismatch", fmt.Errorf("OpenBao alias %q is named %q, expected %q", alias.Status.ID, observed.Name, alias.Spec.Name))
	}
	if observed.MountAccessor != "" && observed.MountAccessor != alias.Spec.MountAccessor {
		return r.fail(ctx, &alias, "AliasIdentityMismatch", fmt.Errorf("OpenBao alias %q uses mount accessor %q, expected %q", alias.Status.ID, observed.MountAccessor, alias.Spec.MountAccessor))
	}
	observeEntityAliasStatus(&alias, observed)
	markReady(&alias.Status.Conditions, alias.Generation, "OpenBao entity alias is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &alias, before, &alias.Status); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Reconciled OpenBao entity alias", "id", alias.Status.ID, "name", alias.Spec.Name, "entityID", alias.Status.CanonicalID)
	return ctrl.Result{RequeueAfter: aliasDriftInterval(&alias)}, nil
}

func (r *OpenBaoEntityAliasReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (AliasClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoEntityAliasReconciler) acquire(ctx context.Context, alias *openbaov1alpha1.OpenBaoEntityAlias, entityID string, apiClient AliasClient) (*openbaoclient.EntityAlias, error) {
	if alias.Status.ID != "" {
		observed, err := apiClient.GetEntityAliasByID(ctx, alias.Status.ID)
		if err != nil {
			if isNotFound(err) {
				alias.Status.ID = ""
				return r.acquire(ctx, alias, entityID, apiClient)
			}
			return nil, err
		}
		return observed, nil
	}

	observed, err := r.findExistingAlias(ctx, alias, apiClient)
	if err != nil {
		return nil, err
	}
	if observed != nil {
		if !aliasCreationPolicy(alias).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao entity alias %q already exists for mount accessor %q; set creationPolicy to Adopt or CreateOrAdopt to manage it", alias.Spec.Name, alias.Spec.MountAccessor)
		}
		if observed.ID == "" {
			return nil, fmt.Errorf("OpenBao returned an entity alias named %q without an ID", alias.Spec.Name)
		}
		alias.Status.ID = observed.ID
		return observed, nil
	}
	if !aliasCreationPolicy(alias).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao entity alias %q does not exist and creationPolicy=%s does not allow creation", alias.Spec.Name, aliasCreationPolicy(alias))
	}

	id, err := apiClient.CreateEntityAlias(ctx, desiredEntityAliasRequest(alias, entityID))
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("OpenBao returned an empty ID when creating entity alias %q", alias.Spec.Name)
	}
	alias.Status.ID = id
	return apiClient.GetEntityAliasByID(ctx, id)
}

func (r *OpenBaoEntityAliasReconciler) findExistingAlias(ctx context.Context, alias *openbaov1alpha1.OpenBaoEntityAlias, apiClient AliasClient) (*openbaoclient.EntityAlias, error) {
	ids, err := apiClient.ListEntityAliasIDs(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var match *openbaoclient.EntityAlias
	for _, id := range ids {
		if id == "" {
			continue
		}
		candidate, err := apiClient.GetEntityAliasByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		if candidate == nil || candidate.Name != alias.Spec.Name || candidate.MountAccessor != alias.Spec.MountAccessor {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("OpenBao returned multiple entity aliases named %q for mount accessor %q", alias.Spec.Name, alias.Spec.MountAccessor)
		}
		match = candidate
	}
	return match, nil
}

func (r *OpenBaoEntityAliasReconciler) reconcileDeletion(ctx context.Context, alias *openbaov1alpha1.OpenBaoEntityAlias) (ctrl.Result, error) {
	if aliasDeletionPolicy(alias) != openbaov1alpha1.DeletionPolicyDelete || alias.Status.ID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, alias)
	}
	connection, err := resolveConnection(ctx, r.Client, alias.Namespace, alias.Spec.ConnectionRef)
	if err != nil {
		if isNotFound(err) {
			return removeFinalizerAfterDependencyLoss(ctx, r.Client, alias, "OpenBaoConnection", err)
		}
		return ctrl.Result{}, err
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		if isNotFound(err) {
			return removeFinalizerAfterDependencyLoss(ctx, r.Client, alias, "OpenBaoConnection credentials", err)
		}
		return ctrl.Result{}, err
	}
	if err := apiClient.DeleteEntityAlias(ctx, alias.Status.ID); err != nil && !isNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, alias)
}

func (r *OpenBaoEntityAliasReconciler) dependencyFailure(ctx context.Context, alias *openbaov1alpha1.OpenBaoEntityAlias, err error) (ctrl.Result, error) {
	before := alias.Status.DeepCopy()
	alias.Status.ObservedGeneration = alias.Generation
	markStalled(&alias.Status.Conditions, alias.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, alias, before, &alias.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoEntityAliasReconciler) fail(ctx context.Context, alias *openbaov1alpha1.OpenBaoEntityAlias, reason string, err error) (ctrl.Result, error) {
	before := alias.Status.DeepCopy()
	alias.Status.ObservedGeneration = alias.Generation
	markError(&alias.Status.Conditions, alias.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, alias, before, &alias.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoEntityAliasReconciler) mapConnectionToAliases(ctx context.Context, obj client.Object) []reconcile.Request {
	var aliases openbaov1alpha1.OpenBaoEntityAliasList
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

func (r *OpenBaoEntityAliasReconciler) mapEntityToAliases(ctx context.Context, obj client.Object) []reconcile.Request {
	var aliases openbaov1alpha1.OpenBaoEntityAliasList
	if err := r.List(ctx, &aliases, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range aliases.Items {
		alias := &aliases.Items[i]
		if alias.Spec.EntityRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(alias)})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenBaoEntityAliasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoEntityAlias{}).
		Named("openbao-openbaoentityalias").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToAliases)).
		Watches(&openbaov1alpha1.OpenBaoEntity{}, handler.EnqueueRequestsFromMapFunc(r.mapEntityToAliases)).
		Complete(r)
}

func desiredEntityAliasRequest(alias *openbaov1alpha1.OpenBaoEntityAlias, entityID string) openbaoclient.EntityAliasRequest {
	return openbaoclient.EntityAliasRequest{
		Name:          alias.Spec.Name,
		MountAccessor: alias.Spec.MountAccessor,
		CanonicalID:   entityID,
	}
}

func entityAliasMatches(desired openbaoclient.EntityAliasRequest, current *openbaoclient.EntityAlias) bool {
	canonicalID := current.CanonicalID
	if canonicalID == "" {
		canonicalID = current.EntityID
	}
	return current.Name == desired.Name && current.MountAccessor == desired.MountAccessor && canonicalID == desired.CanonicalID
}

func observeEntityAliasStatus(alias *openbaov1alpha1.OpenBaoEntityAlias, observed *openbaoclient.EntityAlias) {
	alias.Status.ID = observed.ID
	alias.Status.CanonicalID = observed.CanonicalID
	if alias.Status.CanonicalID == "" {
		alias.Status.CanonicalID = observed.EntityID
	}
	alias.Status.Name = observed.Name
	alias.Status.MountAccessor = observed.MountAccessor
	alias.Status.ObservedGeneration = alias.Generation
}

func aliasDriftInterval(alias *openbaov1alpha1.OpenBaoEntityAlias) time.Duration {
	if alias.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return alias.Spec.DriftDetectionInterval.Duration
}

func aliasCreationPolicy(alias *openbaov1alpha1.OpenBaoEntityAlias) openbaov1alpha1.CreationPolicy {
	if alias.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return alias.Spec.CreationPolicy
}

func aliasDeletionPolicy(alias *openbaov1alpha1.OpenBaoEntityAlias) openbaov1alpha1.DeletionPolicy {
	if alias.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return alias.Spec.DeletionPolicy
}
