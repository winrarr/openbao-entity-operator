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

const defaultKubernetesAuthRoleMount = "kubernetes"

// KubernetesAuthRoleClient is the OpenBao Kubernetes Auth role surface
// reconciled by OpenBaoKubernetesAuthRole.
type KubernetesAuthRoleClient interface {
	GetKubernetesAuthRole(context.Context, string, string) (*openbaoclient.KubernetesAuthRole, error)
	WriteKubernetesAuthRole(context.Context, string, string, openbaoclient.KubernetesAuthRoleRequest) error
	DeleteKubernetesAuthRole(context.Context, string, string) error
}

// OpenBaoKubernetesAuthRoleReconciler reconciles Kubernetes Auth roles.
type OpenBaoKubernetesAuthRoleReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (KubernetesAuthRoleClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaokubernetesauthroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaokubernetesauthroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaokubernetesauthroles/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao Kubernetes Auth role by Kubernetes resource name.
func (r *OpenBaoKubernetesAuthRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var role openbaov1alpha1.OpenBaoKubernetesAuthRole
	if err := r.Get(ctx, req.NamespacedName, &role); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if role.DeletionTimestamp.IsZero() && kubernetesAuthRoleDeletionPolicy(&role).IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &role); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !role.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &role)
	}

	before := role.Status.DeepCopy()
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

	mountPath := kubernetesAuthRoleMountPath(&role)
	observed, err := r.acquire(ctx, &role, apiClient, mountPath)
	if err != nil {
		return r.fail(ctx, &role, "KubernetesAuthRoleAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &role, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no Kubernetes Auth role for %q", role.Name))
	}

	desired, err := desiredKubernetesAuthRoleRequest(&role)
	if err != nil {
		return r.fail(ctx, &role, "InvalidKubernetesAuthRoleSpec", err)
	}
	if !kubernetesAuthRoleMatches(desired, observed) {
		if err := apiClient.WriteKubernetesAuthRole(ctx, mountPath, role.Name, desired); err != nil {
			return r.fail(ctx, &role, "KubernetesAuthRoleWriteFailed", err)
		}
		observed, err = apiClient.GetKubernetesAuthRole(ctx, mountPath, role.Name)
		if err != nil {
			return r.fail(ctx, &role, "KubernetesAuthRoleReadFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &role, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no Kubernetes Auth role after writing %q", role.Name))
		}
	}

	observeKubernetesAuthRoleStatus(&role, observed, mountPath)
	markReady(&role.Status.Conditions, role.Generation, "OpenBao Kubernetes Auth role is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &role, before, &role.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: kubernetesAuthRoleDriftInterval(&role)}, nil
}

func (r *OpenBaoKubernetesAuthRoleReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (KubernetesAuthRoleClient, error) {
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

func (r *OpenBaoKubernetesAuthRoleReconciler) acquire(ctx context.Context, role *openbaov1alpha1.OpenBaoKubernetesAuthRole, apiClient KubernetesAuthRoleClient, mountPath string) (*openbaoclient.KubernetesAuthRole, error) {
	observed, err := apiClient.GetKubernetesAuthRole(ctx, mountPath, role.Name)
	if err == nil {
		if role.Status.Name == "" && !kubernetesAuthRoleCreationPolicy(role).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao Kubernetes Auth role %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", role.Name)
		}
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !kubernetesAuthRoleCreationPolicy(role).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao Kubernetes Auth role %q does not exist and creationPolicy=%s does not allow creation", role.Name, kubernetesAuthRoleCreationPolicy(role))
	}

	desired, err := desiredKubernetesAuthRoleRequest(role)
	if err != nil {
		return nil, err
	}
	if err := apiClient.WriteKubernetesAuthRole(ctx, mountPath, role.Name, desired); err != nil {
		return nil, err
	}
	return apiClient.GetKubernetesAuthRole(ctx, mountPath, role.Name)
}

func (r *OpenBaoKubernetesAuthRoleReconciler) reconcileDeletion(ctx context.Context, role *openbaov1alpha1.OpenBaoKubernetesAuthRole) (ctrl.Result, error) {
	if kubernetesAuthRoleDeletionPolicy(role) != openbaov1alpha1.DeletionPolicyDelete {
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
	if err := apiClient.DeleteKubernetesAuthRole(ctx, kubernetesAuthRoleMountPath(role), role.Name); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, role, before, &role.Status, &role.Status.Conditions, role.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, role)
}

func (r *OpenBaoKubernetesAuthRoleReconciler) dependencyFailure(ctx context.Context, role *openbaov1alpha1.OpenBaoKubernetesAuthRole, err error) (ctrl.Result, error) {
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	markStalled(&role.Status.Conditions, role.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, role, before, &role.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoKubernetesAuthRoleReconciler) fail(ctx context.Context, role *openbaov1alpha1.OpenBaoKubernetesAuthRole, reason string, err error) (ctrl.Result, error) {
	before := role.Status.DeepCopy()
	role.Status.ObservedGeneration = role.Generation
	markError(&role.Status.Conditions, role.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, role, before, &role.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoKubernetesAuthRoleReconciler) mapConnectionToKubernetesAuthRoles(ctx context.Context, obj client.Object) []reconcile.Request {
	var roles openbaov1alpha1.OpenBaoKubernetesAuthRoleList
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

// SetupWithManager sets up the Kubernetes Auth role controller and its connection watch.
func (r *OpenBaoKubernetesAuthRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoKubernetesAuthRole{}).
		Named("openbao-openbaokubernetesauthrole").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToKubernetesAuthRoles)).
		Complete(r)
}

func kubernetesAuthRoleMountPath(role *openbaov1alpha1.OpenBaoKubernetesAuthRole) string {
	if role.Spec.MountPath == "" {
		return defaultKubernetesAuthRoleMount
	}
	return role.Spec.MountPath
}

func desiredKubernetesAuthRoleRequest(role *openbaov1alpha1.OpenBaoKubernetesAuthRole) (openbaoclient.KubernetesAuthRoleRequest, error) {
	names, err := normalizedRoleValues(role.Spec.BoundServiceAccountNames, "boundServiceAccountNames")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	namespaces, err := normalizedRoleValues(role.Spec.BoundServiceAccountNamespaces, "boundServiceAccountNamespaces")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	policies, err := normalizedRoleValues(role.Spec.TokenPolicies, "tokenPolicies")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	tokenTTL, err := durationSeconds(role.Spec.TokenTTL, "tokenTTL")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	tokenMaxTTL, err := durationSeconds(role.Spec.TokenMaxTTL, "tokenMaxTTL")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	tokenPeriod, err := durationSeconds(role.Spec.TokenPeriod, "tokenPeriod")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	tokenExplicitMaxTTL, err := durationSeconds(role.Spec.TokenExplicitMaxTTL, "tokenExplicitMaxTTL")
	if err != nil {
		return openbaoclient.KubernetesAuthRoleRequest{}, err
	}
	var tokenExplicitMaxTTLValue *int64
	if role.Spec.TokenExplicitMaxTTL != nil {
		tokenExplicitMaxTTLValue = &tokenExplicitMaxTTL
	}
	return openbaoclient.KubernetesAuthRoleRequest{
		BoundServiceAccountNames:      names,
		BoundServiceAccountNamespaces: namespaces,
		TokenPolicies:                 policies,
		TokenTTL:                      tokenTTL,
		TokenMaxTTL:                   tokenMaxTTL,
		TokenPeriod:                   tokenPeriod,
		Audience:                      strings.TrimSpace(role.Spec.Audience),
		TokenType:                     strings.TrimSpace(role.Spec.TokenType),
		TokenNumUses:                  role.Spec.TokenNumUses,
		TokenNoDefaultPolicy:          role.Spec.TokenNoDefaultPolicy,
		TokenExplicitMaxTTL:           tokenExplicitMaxTTLValue,
		TokenBoundCIDRs:               normalizedRoleValuesForComparison(role.Spec.TokenBoundCIDRs),
	}, nil
}

func normalizedKubernetesAuthRoleRequest(role *openbaoclient.KubernetesAuthRole) openbaoclient.KubernetesAuthRoleRequest {
	policies := role.EffectivePolicies()
	return openbaoclient.KubernetesAuthRoleRequest{
		BoundServiceAccountNames:      normalizedRoleValuesForComparison(role.BoundServiceAccountNames),
		BoundServiceAccountNamespaces: normalizedRoleValuesForComparison(role.BoundServiceAccountNamespaces),
		TokenPolicies:                 normalizedRoleValuesForComparison(policies),
		TokenTTL:                      role.TokenTTL,
		TokenMaxTTL:                   role.TokenMaxTTL,
		TokenPeriod:                   role.TokenPeriod,
		Audience:                      role.Audience,
		TokenType:                     role.TokenType,
		TokenNumUses:                  &role.TokenNumUses,
		TokenNoDefaultPolicy:          &role.TokenNoDefaultPolicy,
		TokenExplicitMaxTTL:           &role.TokenExplicitMaxTTL,
		TokenBoundCIDRs:               normalizedRoleValuesForComparison(role.TokenBoundCIDRs),
	}
}

func kubernetesAuthRoleMatches(desired openbaoclient.KubernetesAuthRoleRequest, observed *openbaoclient.KubernetesAuthRole) bool {
	normalized := normalizedKubernetesAuthRoleRequest(observed)
	if !reflect.DeepEqual(desired.BoundServiceAccountNames, normalized.BoundServiceAccountNames) ||
		!reflect.DeepEqual(desired.BoundServiceAccountNamespaces, normalized.BoundServiceAccountNamespaces) ||
		!reflect.DeepEqual(desired.TokenPolicies, normalized.TokenPolicies) ||
		desired.TokenTTL != normalized.TokenTTL || desired.TokenMaxTTL != normalized.TokenMaxTTL || desired.TokenPeriod != normalized.TokenPeriod {
		return false
	}
	if desired.Audience != "" && desired.Audience != normalized.Audience {
		return false
	}
	if desired.TokenType != "" && desired.TokenType != normalized.TokenType {
		return false
	}
	if desired.TokenNumUses != nil && (normalized.TokenNumUses == nil || *desired.TokenNumUses != *normalized.TokenNumUses) {
		return false
	}
	if desired.TokenNoDefaultPolicy != nil && (normalized.TokenNoDefaultPolicy == nil || *desired.TokenNoDefaultPolicy != *normalized.TokenNoDefaultPolicy) {
		return false
	}
	if desired.TokenExplicitMaxTTL != nil && (normalized.TokenExplicitMaxTTL == nil || *desired.TokenExplicitMaxTTL != *normalized.TokenExplicitMaxTTL) {
		return false
	}
	return len(desired.TokenBoundCIDRs) == 0 || reflect.DeepEqual(desired.TokenBoundCIDRs, normalized.TokenBoundCIDRs)
}

func observeKubernetesAuthRoleStatus(role *openbaov1alpha1.OpenBaoKubernetesAuthRole, observed *openbaoclient.KubernetesAuthRole, mountPath string) {
	role.Status.MountPath = mountPath
	role.Status.Name = role.Name
	role.Status.ConfigHash = kubernetesAuthRoleConfigHash(observed)
	role.Status.ObservedGeneration = role.Generation
}

func kubernetesAuthRoleConfigHash(role *openbaoclient.KubernetesAuthRole) string {
	data, err := json.Marshal(normalizedKubernetesAuthRoleRequest(role))
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func normalizedRoleValues(values []string, field string) ([]string, error) {
	result := normalizedRoleValuesForComparison(values)
	if slices.Contains(result, "") {
		return nil, fmt.Errorf("%s must not contain empty values", field)
	}
	return result, nil
}

func normalizedRoleValuesForComparison(values []string) []string {
	result := append([]string{}, values...)
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func durationSeconds(duration *metav1.Duration, field string) (int64, error) {
	if duration == nil || duration.Duration == 0 {
		return 0, nil
	}
	if duration.Duration < 0 {
		return 0, fmt.Errorf("%s must not be negative", field)
	}
	if duration.Duration%time.Second != 0 {
		return 0, fmt.Errorf("%s must be a whole number of seconds", field)
	}
	return int64(duration.Duration / time.Second), nil
}

func kubernetesAuthRoleDriftInterval(role *openbaov1alpha1.OpenBaoKubernetesAuthRole) time.Duration {
	if role.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return role.Spec.DriftDetectionInterval.Duration
}

func kubernetesAuthRoleCreationPolicy(role *openbaov1alpha1.OpenBaoKubernetesAuthRole) openbaov1alpha1.CreationPolicy {
	if role.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return role.Spec.CreationPolicy
}

func kubernetesAuthRoleDeletionPolicy(role *openbaov1alpha1.OpenBaoKubernetesAuthRole) openbaov1alpha1.DeletionPolicy {
	if role.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return role.Spec.DeletionPolicy
}
