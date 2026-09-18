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
	"fmt"
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

// PasswordPolicyClient is the OpenBao password-policy surface reconciled by OpenBaoPasswordPolicy.
type PasswordPolicyClient interface {
	GetPasswordPolicy(context.Context, string) (*openbaoclient.PasswordPolicy, error)
	WritePasswordPolicy(context.Context, string, string) error
	DeletePasswordPolicy(context.Context, string) error
}

// OpenBaoPasswordPolicyReconciler reconciles OpenBaoPasswordPolicy resources.
type OpenBaoPasswordPolicyReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (PasswordPolicyClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopasswordpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopasswordpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopasswordpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao password policy by Kubernetes resource name.
func (r *OpenBaoPasswordPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var policy openbaov1alpha1.OpenBaoPasswordPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if policy.DeletionTimestamp.IsZero() && passwordPolicyDeletionPolicy(&policy).IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &policy); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !policy.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &policy)
	}

	before := policy.Status.DeepCopy()
	policy.Status.ObservedGeneration = policy.Generation
	connection, err := resolveConnection(ctx, r.Client, policy.Namespace, policy.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &policy, fmt.Errorf("read OpenBaoConnection %s/%s: %w", policy.Namespace, policy.Spec.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &policy, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &policy, err)
	}
	observed, err := r.acquire(ctx, &policy, apiClient)
	if err != nil {
		return r.fail(ctx, &policy, "PasswordPolicyAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &policy, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no password policy for %q", policy.Name))
	}
	if observed.Rules != policy.Spec.Rules {
		if err := apiClient.WritePasswordPolicy(ctx, policy.Name, policy.Spec.Rules); err != nil {
			return r.fail(ctx, &policy, "PasswordPolicyWriteFailed", err)
		}
		observed, err = apiClient.GetPasswordPolicy(ctx, policy.Name)
		if err != nil {
			return r.fail(ctx, &policy, "PasswordPolicyReadFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &policy, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no password policy after writing %q", policy.Name))
		}
	}

	policy.Status.Name = policy.Name
	policy.Status.RulesHash = passwordPolicyRulesHash(observed.Rules)
	markReady(&policy.Status.Conditions, policy.Generation, "OpenBao password policy is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &policy, before, &policy.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: passwordPolicyDriftInterval(&policy)}, nil
}

func (r *OpenBaoPasswordPolicyReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (PasswordPolicyClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoPasswordPolicyReconciler) acquire(ctx context.Context, policy *openbaov1alpha1.OpenBaoPasswordPolicy, apiClient PasswordPolicyClient) (*openbaoclient.PasswordPolicy, error) {
	observed, err := apiClient.GetPasswordPolicy(ctx, policy.Name)
	if err == nil {
		if policy.Status.Name == "" && !passwordPolicyCreationPolicy(policy).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao password policy %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", policy.Name)
		}
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !passwordPolicyCreationPolicy(policy).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao password policy %q does not exist and creationPolicy=%s does not allow creation", policy.Name, passwordPolicyCreationPolicy(policy))
	}
	if err := apiClient.WritePasswordPolicy(ctx, policy.Name, policy.Spec.Rules); err != nil {
		return nil, err
	}
	return apiClient.GetPasswordPolicy(ctx, policy.Name)
}

func (r *OpenBaoPasswordPolicyReconciler) reconcileDeletion(ctx context.Context, policy *openbaov1alpha1.OpenBaoPasswordPolicy) (ctrl.Result, error) {
	if passwordPolicyDeletionPolicy(policy) != openbaov1alpha1.DeletionPolicyDelete {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, policy)
	}
	before := policy.Status.DeepCopy()
	policy.Status.ObservedGeneration = policy.Generation
	connection, err := resolveConnection(ctx, r.Client, policy.Namespace, policy.Spec.ConnectionRef)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, policy, before, &policy.Status, &policy.Status.Conditions, policy.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, policy, "OpenBaoConnection", err))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, policy, before, &policy.Status, &policy.Status.Conditions, policy.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, policy, "OpenBaoConnection credentials", err))
	}
	if err := apiClient.DeletePasswordPolicy(ctx, policy.Name); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, policy, before, &policy.Status, &policy.Status.Conditions, policy.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, policy)
}

func (r *OpenBaoPasswordPolicyReconciler) dependencyFailure(ctx context.Context, policy *openbaov1alpha1.OpenBaoPasswordPolicy, err error) (ctrl.Result, error) {
	before := policy.Status.DeepCopy()
	policy.Status.ObservedGeneration = policy.Generation
	markStalled(&policy.Status.Conditions, policy.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, policy, before, &policy.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoPasswordPolicyReconciler) fail(ctx context.Context, policy *openbaov1alpha1.OpenBaoPasswordPolicy, reason string, err error) (ctrl.Result, error) {
	before := policy.Status.DeepCopy()
	policy.Status.ObservedGeneration = policy.Generation
	markError(&policy.Status.Conditions, policy.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, policy, before, &policy.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoPasswordPolicyReconciler) mapConnectionToPasswordPolicies(ctx context.Context, obj client.Object) []reconcile.Request {
	var policies openbaov1alpha1.OpenBaoPasswordPolicyList
	if err := r.List(ctx, &policies, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range policies.Items {
		policy := &policies.Items[i]
		if policy.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(policy)})
		}
	}
	return requests
}

// SetupWithManager sets up the password-policy controller.
func (r *OpenBaoPasswordPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoPasswordPolicy{}).
		Named("openbao-openbaopasswordpolicy").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToPasswordPolicies)).
		Complete(r)
}

func passwordPolicyRulesHash(rules string) string {
	digest := sha256.Sum256([]byte(rules))
	return hex.EncodeToString(digest[:])
}

func passwordPolicyDriftInterval(policy *openbaov1alpha1.OpenBaoPasswordPolicy) time.Duration {
	if policy.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return policy.Spec.DriftDetectionInterval.Duration
}

func passwordPolicyCreationPolicy(policy *openbaov1alpha1.OpenBaoPasswordPolicy) openbaov1alpha1.CreationPolicy {
	if policy.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return policy.Spec.CreationPolicy
}

func passwordPolicyDeletionPolicy(policy *openbaov1alpha1.OpenBaoPasswordPolicy) openbaov1alpha1.DeletionPolicy {
	if policy.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return policy.Spec.DeletionPolicy
}
