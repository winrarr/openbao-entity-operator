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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

// TokenRoleClient is the OpenBao token-role surface reconciled by OpenBaoTokenRole.
type TokenRoleClient interface {
	GetTokenRole(context.Context, string) (*openbaoclient.TokenRole, error)
	WriteTokenRole(context.Context, string, openbaoclient.TokenRoleRequest) error
	DeleteTokenRole(context.Context, string) error
}

// OpenBaoTokenRoleReconciler reconciles OpenBaoTokenRole resources.
type OpenBaoTokenRoleReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (TokenRoleClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaotokenroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaotokenroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaotokenroles/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao token role by Kubernetes resource name.
func (r *OpenBaoTokenRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var role openbaov1alpha1.OpenBaoTokenRole
	if err := r.Get(ctx, req.NamespacedName, &role); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := role.Status.DeepCopy()
	if role.DeletionTimestamp.IsZero() && tokenRoleDeletionPolicy(&role).IsDelete() {
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
	if !conditionIsReady(connection) {
		return r.dependencyFailure(ctx, &role, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}

	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &role, err)
	}
	observed, err := r.acquire(ctx, &role, apiClient)
	if err != nil {
		return r.fail(ctx, &role, "TokenRoleAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &role, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no token role for %q", role.Name))
	}
	desired, err := desiredTokenRoleRequest(&role)
	if err != nil {
		return r.fail(ctx, &role, "InvalidTokenRoleSpec", err)
	}
	if !tokenRoleMatches(desired, observed) {
		if err := apiClient.WriteTokenRole(ctx, role.Name, desired); err != nil {
			return r.fail(ctx, &role, "TokenRoleWriteFailed", err)
		}
		observed, err = apiClient.GetTokenRole(ctx, role.Name)
		if err != nil {
			return r.fail(ctx, &role, "TokenRoleReadFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &role, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no token role after writing %q", role.Name))
		}
	}

	role.Status.Name = role.Name
	role.Status.ConfigHash = tokenRoleConfigHash(observed)
	markReady(&role.Status.Conditions, role.Generation, "OpenBao token role is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &role, before, &role.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: tokenRoleDriftInterval(&role)}, nil
}

func (r *OpenBaoTokenRoleReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (TokenRoleClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoTokenRoleReconciler) acquire(ctx context.Context, role *openbaov1alpha1.OpenBaoTokenRole, apiClient TokenRoleClient) (*openbaoclient.TokenRole, error) {
	observed, err := apiClient.GetTokenRole(ctx, role.Name)
	if err == nil {
		if !tokenRoleCreationPolicy(role).AllowsAdoption() && role.Status.Name == "" {
			return nil, fmt.Errorf("OpenBao token role %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", role.Name)
		}
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !tokenRoleCreationPolicy(role).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao token role %q does not exist and creationPolicy=%s does not allow creation", role.Name, tokenRoleCreationPolicy(role))
	}
	desired, err := desiredTokenRoleRequest(role)
	if err != nil {
		return nil, err
	}
	if err := apiClient.WriteTokenRole(ctx, role.Name, desired); err != nil {
		return nil, err
	}
	return apiClient.GetTokenRole(ctx, role.Name)
}

func (r *OpenBaoTokenRoleReconciler) reconcileDeletion(ctx context.Context, role *openbaov1alpha1.OpenBaoTokenRole) (ctrl.Result, error) {
	if tokenRoleDeletionPolicy(role) != openbaov1alpha1.DeletionPolicyDelete {
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
	if err := apiClient.DeleteTokenRole(ctx, role.Name); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, role, before, &role.Status, &role.Status.Conditions, role.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, role)
}

func (r *OpenBaoTokenRoleReconciler) dependencyFailure(ctx context.Context, role *openbaov1alpha1.OpenBaoTokenRole, err error) (ctrl.Result, error) {
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	markStalled(&role.Status.Conditions, role.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, role, before, &role.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoTokenRoleReconciler) fail(ctx context.Context, role *openbaov1alpha1.OpenBaoTokenRole, reason string, err error) (ctrl.Result, error) {
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	markError(&role.Status.Conditions, role.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, role, before, &role.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoTokenRoleReconciler) mapConnectionToTokenRoles(ctx context.Context, obj client.Object) []reconcile.Request {
	var roles openbaov1alpha1.OpenBaoTokenRoleList
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

// SetupWithManager sets up the controller with the Manager.
func (r *OpenBaoTokenRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoTokenRole{}).
		Named("openbao-openbaotokenrole").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToTokenRoles)).
		Complete(r)
}

func desiredTokenRoleRequest(role *openbaov1alpha1.OpenBaoTokenRole) (openbaoclient.TokenRoleRequest, error) {
	request := openbaoclient.TokenRoleRequest{}
	var err error
	request.AllowedEntityAliases, err = normalizedTokenRoleValues(role.Spec.AllowedEntityAliases, "allowedEntityAliases")
	if err != nil {
		return request, err
	}
	request.AllowedPolicies, err = normalizedTokenRoleValues(role.Spec.AllowedPolicies, "allowedPolicies")
	if err != nil {
		return request, err
	}
	request.AllowedPoliciesGlob, err = normalizedTokenRoleValues(role.Spec.AllowedPoliciesGlob, "allowedPoliciesGlob")
	if err != nil {
		return request, err
	}
	request.DisallowedPolicies, err = normalizedTokenRoleValues(role.Spec.DisallowedPolicies, "disallowedPolicies")
	if err != nil {
		return request, err
	}
	request.DisallowedPoliciesGlob, err = normalizedTokenRoleValues(role.Spec.DisallowedPoliciesGlob, "disallowedPoliciesGlob")
	if err != nil {
		return request, err
	}
	request.TokenBoundCIDRs, err = normalizedTokenRoleValues(role.Spec.TokenBoundCIDRs, "tokenBoundCIDRs")
	if err != nil {
		return request, err
	}
	request.TokenExplicitMaxTTL, err = optionalDurationSeconds(role.Spec.TokenExplicitMaxTTL, "tokenExplicitMaxTTL")
	if err != nil {
		return request, err
	}
	request.TokenPeriod, err = optionalDurationSeconds(role.Spec.TokenPeriod, "tokenPeriod")
	if err != nil {
		return request, err
	}
	request.TokenNumUses = role.Spec.TokenNumUses
	request.TokenNoDefaultPolicy = role.Spec.TokenNoDefaultPolicy
	request.Orphan = role.Spec.Orphan
	request.Renewable = role.Spec.Renewable
	request.TokenType = strings.TrimSpace(role.Spec.TokenType)
	request.PathSuffix = strings.TrimSpace(role.Spec.PathSuffix)
	return request, nil
}

func tokenRoleMatches(desired openbaoclient.TokenRoleRequest, observed *openbaoclient.TokenRole) bool {
	normalized := normalizedTokenRole(observed)
	if !reflect.DeepEqual(desired.AllowedEntityAliases, normalized.AllowedEntityAliases) ||
		!reflect.DeepEqual(desired.AllowedPolicies, normalized.AllowedPolicies) ||
		!reflect.DeepEqual(desired.AllowedPoliciesGlob, normalized.AllowedPoliciesGlob) ||
		!reflect.DeepEqual(desired.DisallowedPolicies, normalized.DisallowedPolicies) ||
		!reflect.DeepEqual(desired.DisallowedPoliciesGlob, normalized.DisallowedPoliciesGlob) ||
		!reflect.DeepEqual(desired.TokenBoundCIDRs, normalized.TokenBoundCIDRs) {
		return false
	}
	return optionalIntMatches(desired.TokenExplicitMaxTTL, normalized.TokenExplicitMaxTTL) &&
		optionalIntMatches(desired.TokenPeriod, normalized.TokenPeriod) &&
		optionalIntMatches(desired.TokenNumUses, normalized.TokenNumUses) &&
		optionalBoolMatches(desired.TokenNoDefaultPolicy, normalized.TokenNoDefaultPolicy) &&
		optionalBoolMatches(desired.Orphan, normalized.Orphan) &&
		optionalBoolMatches(desired.Renewable, normalized.Renewable) &&
		(desired.TokenType == "" || desired.TokenType == normalized.TokenType) &&
		(desired.PathSuffix == "" || desired.PathSuffix == normalized.PathSuffix)
}

func normalizedTokenRole(role *openbaoclient.TokenRole) openbaoclient.TokenRoleRequest {
	return openbaoclient.TokenRoleRequest{
		AllowedEntityAliases:   normalizedTokenRoleValuesForComparison(role.AllowedEntityAliases),
		AllowedPolicies:        normalizedTokenRoleValuesForComparison(role.AllowedPolicies),
		AllowedPoliciesGlob:    normalizedTokenRoleValuesForComparison(role.AllowedPoliciesGlob),
		DisallowedPolicies:     normalizedTokenRoleValuesForComparison(role.DisallowedPolicies),
		DisallowedPoliciesGlob: normalizedTokenRoleValuesForComparison(role.DisallowedPoliciesGlob),
		TokenBoundCIDRs:        normalizedTokenRoleValuesForComparison(role.TokenBoundCIDRs),
		TokenExplicitMaxTTL:    role.TokenExplicitMaxTTL,
		TokenPeriod:            role.TokenPeriod,
		TokenNumUses:           role.TokenNumUses,
		TokenNoDefaultPolicy:   role.TokenNoDefaultPolicy,
		TokenType:              role.TokenType,
		Orphan:                 role.Orphan,
		Renewable:              role.Renewable,
		PathSuffix:             role.PathSuffix,
	}
}

func tokenRoleConfigHash(role *openbaoclient.TokenRole) string {
	data, err := json.Marshal(normalizedTokenRole(role))
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func optionalDurationSeconds(duration *metav1.Duration, field string) (*int64, error) {
	if duration == nil {
		return nil, nil
	}
	seconds, err := durationSeconds(duration, field)
	if err != nil {
		return nil, err
	}
	return &seconds, nil
}

func optionalIntMatches(desired, observed *int64) bool {
	return desired == nil || (observed != nil && *desired == *observed)
}

func optionalBoolMatches(desired, observed *bool) bool {
	return desired == nil || (observed != nil && *desired == *observed)
}

func normalizedTokenRoleValues(values []string, field string) ([]string, error) {
	result := normalizedTokenRoleValuesForComparison(values)
	if slices.Contains(result, "") {
		return nil, fmt.Errorf("%s must not contain empty values", field)
	}
	return result, nil
}

func normalizedTokenRoleValuesForComparison(values []string) []string {
	result := append([]string(nil), values...)
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func tokenRoleDriftInterval(role *openbaov1alpha1.OpenBaoTokenRole) time.Duration {
	if role.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return role.Spec.DriftDetectionInterval.Duration
}

func tokenRoleCreationPolicy(role *openbaov1alpha1.OpenBaoTokenRole) openbaov1alpha1.CreationPolicy {
	if role.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return role.Spec.CreationPolicy
}

func tokenRoleDeletionPolicy(role *openbaov1alpha1.OpenBaoTokenRole) openbaov1alpha1.DeletionPolicy {
	if role.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return role.Spec.DeletionPolicy
}

func conditionIsReady(connection *openbaov1alpha1.OpenBaoConnection) bool {
	return meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady)
}
