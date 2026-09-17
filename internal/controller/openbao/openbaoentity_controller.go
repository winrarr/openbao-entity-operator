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

// EntityClient is the OpenBao entity surface reconciled by OpenBaoEntity.
type EntityClient interface {
	GetEntityByID(context.Context, string) (*openbaoclient.Entity, error)
	GetEntityByName(context.Context, string) (*openbaoclient.Entity, error)
	CreateEntity(context.Context, openbaoclient.EntityRequest) (string, error)
	UpdateEntity(context.Context, string, openbaoclient.EntityRequest) (*openbaoclient.Entity, error)
	DeleteEntity(context.Context, string) error
}

// OpenBaoEntityReconciler reconciles an OpenBaoEntity object.
type OpenBaoEntityReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (EntityClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoentities/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *OpenBaoEntityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("openbao-entity")
	var entity openbaov1alpha1.OpenBaoEntity
	if err := r.Get(ctx, req.NamespacedName, &entity); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := entity.Status.DeepCopy()
	if entity.DeletionTimestamp.IsZero() && deletionPolicy(&entity).IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &entity); err != nil {
			return ctrl.Result{}, err
		}
	}

	if !entity.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &entity)
	}

	entity.Status.ObservedGeneration = entity.Generation

	connection, err := resolveConnection(ctx, r.Client, entity.Namespace, entity.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &entity, fmt.Errorf("read OpenBaoConnection %s/%s: %w", entity.Namespace, entity.Spec.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &entity, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}

	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &entity, err)
	}

	observed, err := r.acquire(ctx, &entity, apiClient)
	if err != nil {
		return r.fail(ctx, &entity, "EntityAcquireFailed", err)
	}

	desired := desiredEntityRequest(&entity)
	if !entityMatches(desired, observed) {
		observed, err = apiClient.UpdateEntity(ctx, entity.Status.ID, desired)
		if err != nil {
			return r.fail(ctx, &entity, "EntityUpdateFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &entity, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no entity after updating %q", entity.Status.ID))
		}
		observed, err = apiClient.GetEntityByID(ctx, entity.Status.ID)
		if err != nil {
			return r.fail(ctx, &entity, "EntityReadFailed", err)
		}
	}

	if observed.Name != "" && observed.Name != entity.Name {
		return r.fail(ctx, &entity, "EntityIdentityMismatch", fmt.Errorf("OpenBao entity %q is named %q, expected %q", entity.Status.ID, observed.Name, entity.Name))
	}
	observeEntityStatus(&entity, observed)
	markReady(&entity.Status.Conditions, entity.Generation, "OpenBao entity is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &entity, before, &entity.Status); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Reconciled OpenBao entity", "id", entity.Status.ID, "name", entity.Name)
	return ctrl.Result{RequeueAfter: entityDriftInterval(&entity)}, nil

}

func (r *OpenBaoEntityReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (EntityClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoEntityReconciler) acquire(ctx context.Context, entity *openbaov1alpha1.OpenBaoEntity, apiClient EntityClient) (*openbaoclient.Entity, error) {
	if entity.Status.ID != "" {
		observed, err := apiClient.GetEntityByID(ctx, entity.Status.ID)
		if err != nil {
			if isNotFound(err) {
				entity.Status.ID = ""
				return r.acquire(ctx, entity, apiClient)
			}
			return nil, err
		}
		return observed, nil
	}

	observed, err := apiClient.GetEntityByName(ctx, entity.Name)
	if err == nil {
		if !creationPolicy(entity).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao entity %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", entity.Name)
		}
		if observed.ID == "" {
			return nil, fmt.Errorf("OpenBao returned an entity named %q without an ID", entity.Name)
		}
		entity.Status.ID = observed.ID
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !creationPolicy(entity).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao entity %q does not exist and creationPolicy=%s does not allow creation", entity.Name, creationPolicy(entity))
	}

	id, err := apiClient.CreateEntity(ctx, desiredEntityRequest(entity))
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("OpenBao returned an empty ID when creating entity %q", entity.Name)
	}
	entity.Status.ID = id
	return apiClient.GetEntityByID(ctx, id)
}

func (r *OpenBaoEntityReconciler) reconcileDeletion(ctx context.Context, entity *openbaov1alpha1.OpenBaoEntity) (ctrl.Result, error) {
	if deletionPolicy(entity) != openbaov1alpha1.DeletionPolicyDelete || entity.Status.ID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, entity)
	}
	before := entity.Status.DeepCopy()
	entity.Status.ObservedGeneration = entity.Generation
	connection, err := resolveConnection(ctx, r.Client, entity.Namespace, entity.Spec.ConnectionRef)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, entity, before, &entity.Status, &entity.Status.Conditions, entity.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, entity, "OpenBaoConnection", err))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, entity, before, &entity.Status, &entity.Status.Conditions, entity.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, entity, "OpenBaoConnection credentials", err))
	}
	if err := apiClient.DeleteEntity(ctx, entity.Status.ID); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, entity, before, &entity.Status, &entity.Status.Conditions, entity.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, entity)
}

func (r *OpenBaoEntityReconciler) dependencyFailure(ctx context.Context, entity *openbaov1alpha1.OpenBaoEntity, err error) (ctrl.Result, error) {
	before := entity.Status.DeepCopy()
	entity.Status.ObservedGeneration = entity.Generation
	markStalled(&entity.Status.Conditions, entity.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, entity, before, &entity.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoEntityReconciler) fail(ctx context.Context, entity *openbaov1alpha1.OpenBaoEntity, reason string, err error) (ctrl.Result, error) {
	before := entity.Status.DeepCopy()
	entity.Status.ObservedGeneration = entity.Generation
	markError(&entity.Status.Conditions, entity.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, entity, before, &entity.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoEntityReconciler) mapConnectionToEntities(ctx context.Context, obj client.Object) []reconcile.Request {
	var entities openbaov1alpha1.OpenBaoEntityList
	if err := r.List(ctx, &entities, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range entities.Items {
		entity := &entities.Items[i]
		if entity.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(entity)})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenBaoEntityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoEntity{}).
		Named("openbao-openbaoentity").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToEntities)).
		Complete(r)
}
