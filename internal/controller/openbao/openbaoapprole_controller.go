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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const defaultAppRoleResourceMount = "approle"

// AppRoleClient is the OpenBao AppRole role surface reconciled by OpenBaoAppRole.
type AppRoleClient interface {
	GetAppRole(context.Context, string, string) (*openbaoclient.AppRole, error)
	WriteAppRole(context.Context, string, string, openbaoclient.AppRoleRequest) error
	DeleteAppRole(context.Context, string, string) error
}

// OpenBaoAppRoleReconciler reconciles OpenBaoAppRole resources.
type OpenBaoAppRoleReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (AppRoleClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoapproles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoapproles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoapproles/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao AppRole configuration.
func (r *OpenBaoAppRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var role openbaov1alpha1.OpenBaoAppRole
	if err := r.Get(ctx, req.NamespacedName, &role); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := role.Status.DeepCopy()
	if role.DeletionTimestamp.IsZero() && appRoleDeletionPolicy(&role).IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &role); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !role.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &role)
	}

	role.Status.ObservedGeneration = role.Generation
	connection, err := resolveConnection(ctx, r.Client, role.Namespace, role.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &role, fmt.Errorf("read OpenBaoConnection %s/%s: %w", role.Namespace, role.Spec.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &role, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &role, err)
	}
	mountPath := appRoleMountPath(&role)
	observed, err := r.acquire(ctx, &role, mountPath, apiClient)
	if err != nil {
		return r.fail(ctx, &role, "AppRoleAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &role, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no AppRole %q", role.Name))
	}
	desired, err := desiredAppRoleRequest(&role)
	if err != nil {
		return r.fail(ctx, &role, "InvalidAppRoleSpec", err)
	}
	if !appRoleMatches(desired, observed) {
		if err := apiClient.WriteAppRole(ctx, mountPath, role.Name, desired); err != nil {
			return r.fail(ctx, &role, "AppRoleWriteFailed", err)
		}
		observed, err = apiClient.GetAppRole(ctx, mountPath, role.Name)
		if err != nil {
			return r.fail(ctx, &role, "AppRoleReadFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &role, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no AppRole after writing %q", role.Name))
		}
	}

	role.Status.MountPath = mountPath
	role.Status.Name = role.Name
	role.Status.ConfigHash = appRoleConfigHash(observed)
	markReady(&role.Status.Conditions, role.Generation, "OpenBao AppRole is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &role, before, &role.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: appRoleDriftInterval(&role)}, nil
}

func (r *OpenBaoAppRoleReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (AppRoleClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoAppRoleReconciler) acquire(ctx context.Context, role *openbaov1alpha1.OpenBaoAppRole, mountPath string, apiClient AppRoleClient) (*openbaoclient.AppRole, error) {
	observed, err := apiClient.GetAppRole(ctx, mountPath, role.Name)
	if err == nil {
		if role.Status.Name == "" && !appRoleCreationPolicy(role).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao AppRole %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", role.Name)
		}
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !appRoleCreationPolicy(role).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao AppRole %q does not exist and creationPolicy=%s does not allow creation", role.Name, appRoleCreationPolicy(role))
	}
	desired, err := desiredAppRoleRequest(role)
	if err != nil {
		return nil, err
	}
	if err := apiClient.WriteAppRole(ctx, mountPath, role.Name, desired); err != nil {
		return nil, err
	}
	return apiClient.GetAppRole(ctx, mountPath, role.Name)
}

func (r *OpenBaoAppRoleReconciler) reconcileDeletion(ctx context.Context, role *openbaov1alpha1.OpenBaoAppRole) (ctrl.Result, error) {
	if appRoleDeletionPolicy(role) != openbaov1alpha1.DeletionPolicyDelete {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, role)
	}
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	connection, err := resolveConnection(ctx, r.Client, role.Namespace, role.Spec.ConnectionRef)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, role, before, &role.Status, &role.Status.Conditions, role.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, role, "OpenBaoConnection", err))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, role, before, &role.Status, &role.Status.Conditions, role.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, role, "OpenBaoConnection credentials", err))
	}
	if err := apiClient.DeleteAppRole(ctx, appRoleMountPath(role), role.Name); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, role, before, &role.Status, &role.Status.Conditions, role.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, role)
}

func (r *OpenBaoAppRoleReconciler) dependencyFailure(ctx context.Context, role *openbaov1alpha1.OpenBaoAppRole, err error) (ctrl.Result, error) {
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	markStalled(&role.Status.Conditions, role.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, role, before, &role.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoAppRoleReconciler) fail(ctx context.Context, role *openbaov1alpha1.OpenBaoAppRole, reason string, err error) (ctrl.Result, error) {
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	markError(&role.Status.Conditions, role.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, role, before, &role.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoAppRoleReconciler) mapConnectionToAppRoles(ctx context.Context, obj client.Object) []reconcile.Request {
	var roles openbaov1alpha1.OpenBaoAppRoleList
	if err := r.List(ctx, &roles, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range roles.Items {
		role := &roles.Items[i]
		if role.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(role)})
		}
	}
	return requests
}

// SetupWithManager sets up the AppRole controller.
func (r *OpenBaoAppRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoAppRole{}).
		Named("openbao-openbaoapprole").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToAppRoles)).
		Complete(r)
}

func desiredAppRoleRequest(role *openbaov1alpha1.OpenBaoAppRole) (openbaoclient.AppRoleRequest, error) {
	request := openbaoclient.AppRoleRequest{}
	var err error
	request.BindSecretID = role.Spec.BindSecretID
	request.LocalSecretIDs = role.Spec.LocalSecretIDs
	request.SecretIDBoundCIDRs, err = normalizedAppRoleValues(role.Spec.SecretIDBoundCIDRs, "secretIDBoundCIDRs")
	if err != nil {
		return request, err
	}
	request.TokenBoundCIDRs, err = normalizedAppRoleValues(role.Spec.TokenBoundCIDRs, "tokenBoundCIDRs")
	if err != nil {
		return request, err
	}
	request.SecretIDNumUses = role.Spec.SecretIDNumUses
	request.TokenNumUses = role.Spec.TokenNumUses
	request.TokenNoDefaultPolicy = role.Spec.TokenNoDefaultPolicy
	request.TokenType = strings.TrimSpace(role.Spec.TokenType)
	request.TokenPolicies, err = normalizedAppRoleValues(role.Spec.TokenPolicies, "tokenPolicies")
	if err != nil {
		return request, err
	}
	request.SecretIDTTL, err = optionalDurationSeconds(role.Spec.SecretIDTTL, "secretIDTTL")
	if err != nil {
		return request, err
	}
	request.TokenExplicitMaxTTL, err = optionalDurationSeconds(role.Spec.TokenExplicitMaxTTL, "tokenExplicitMaxTTL")
	if err != nil {
		return request, err
	}
	request.TokenMaxTTL, err = optionalDurationSeconds(role.Spec.TokenMaxTTL, "tokenMaxTTL")
	if err != nil {
		return request, err
	}
	request.TokenPeriod, err = optionalDurationSeconds(role.Spec.TokenPeriod, "tokenPeriod")
	if err != nil {
		return request, err
	}
	request.TokenTTL, err = optionalDurationSeconds(role.Spec.TokenTTL, "tokenTTL")
	return request, err
}

func appRoleMatches(desired openbaoclient.AppRoleRequest, observed *openbaoclient.AppRole) bool {
	normalized := normalizedAppRole(observed)
	return optionalBoolMatches(desired.BindSecretID, normalized.BindSecretID) &&
		optionalBoolMatches(desired.LocalSecretIDs, normalized.LocalSecretIDs) &&
		optionalStringSliceMatches(desired.SecretIDBoundCIDRs, normalized.SecretIDBoundCIDRs) &&
		optionalStringSliceMatches(desired.TokenBoundCIDRs, normalized.TokenBoundCIDRs) &&
		optionalIntMatches(desired.SecretIDNumUses, normalized.SecretIDNumUses) &&
		optionalIntMatches(desired.SecretIDTTL, normalized.SecretIDTTL) &&
		optionalIntMatches(desired.TokenExplicitMaxTTL, normalized.TokenExplicitMaxTTL) &&
		optionalIntMatches(desired.TokenMaxTTL, normalized.TokenMaxTTL) &&
		optionalBoolMatches(desired.TokenNoDefaultPolicy, normalized.TokenNoDefaultPolicy) &&
		optionalIntMatches(desired.TokenNumUses, normalized.TokenNumUses) &&
		optionalIntMatches(desired.TokenPeriod, normalized.TokenPeriod) &&
		optionalStringSliceMatches(desired.TokenPolicies, normalized.TokenPolicies) &&
		optionalIntMatches(desired.TokenTTL, normalized.TokenTTL) &&
		(desired.TokenType == "" || desired.TokenType == normalized.TokenType)
}

func normalizedAppRole(role *openbaoclient.AppRole) openbaoclient.AppRoleRequest {
	return openbaoclient.AppRoleRequest{
		BindSecretID:         role.BindSecretID,
		LocalSecretIDs:       role.LocalSecretIDs,
		SecretIDBoundCIDRs:   normalizedAppRoleValuesForComparison(role.SecretIDBoundCIDRs),
		SecretIDNumUses:      role.SecretIDNumUses,
		SecretIDTTL:          role.SecretIDTTL,
		TokenBoundCIDRs:      normalizedAppRoleValuesForComparison(role.TokenBoundCIDRs),
		TokenExplicitMaxTTL:  role.TokenExplicitMaxTTL,
		TokenMaxTTL:          role.TokenMaxTTL,
		TokenNoDefaultPolicy: role.TokenNoDefaultPolicy,
		TokenNumUses:         role.TokenNumUses,
		TokenPeriod:          role.TokenPeriod,
		TokenPolicies:        normalizedAppRoleValuesForComparison(role.EffectivePolicies()),
		TokenTTL:             role.TokenTTL,
		TokenType:            role.TokenType,
	}
}

func appRoleConfigHash(role *openbaoclient.AppRole) string {
	data, err := json.Marshal(normalizedAppRole(role))
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func optionalStringSliceMatches(desired, observed []string) bool {
	return len(desired) == 0 || reflect.DeepEqual(desired, observed)
}

func normalizedAppRoleValues(values []string, field string) ([]string, error) {
	result := normalizedAppRoleValuesForComparison(values)
	if slices.Contains(result, "") {
		return nil, fmt.Errorf("%s must not contain empty values", field)
	}
	return result, nil
}

func normalizedAppRoleValuesForComparison(values []string) []string {
	result := append([]string(nil), values...)
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func appRoleMountPath(role *openbaov1alpha1.OpenBaoAppRole) string {
	if role.Spec.MountPath == "" {
		return defaultAppRoleResourceMount
	}
	return role.Spec.MountPath
}

func appRoleDriftInterval(role *openbaov1alpha1.OpenBaoAppRole) time.Duration {
	if role.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return role.Spec.DriftDetectionInterval.Duration
}

func appRoleCreationPolicy(role *openbaov1alpha1.OpenBaoAppRole) openbaov1alpha1.CreationPolicy {
	if role.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return role.Spec.CreationPolicy
}

func appRoleDeletionPolicy(role *openbaov1alpha1.OpenBaoAppRole) openbaov1alpha1.DeletionPolicy {
	if role.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return role.Spec.DeletionPolicy
}
