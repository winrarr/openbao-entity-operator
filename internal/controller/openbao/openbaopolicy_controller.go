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

// PolicyClient is the OpenBao ACL policy surface reconciled by OpenBaoPolicy.
type PolicyClient interface {
	GetPolicy(context.Context, string) (*openbaoclient.Policy, error)
	WritePolicy(context.Context, string, openbaoclient.PolicyRequest) error
	DeletePolicy(context.Context, string) error
}

// OpenBaoPolicyReconciler reconciles OpenBaoPolicy resources.
type OpenBaoPolicyReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (PolicyClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao ACL policy by Kubernetes resource name.
func (r *OpenBaoPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var policy openbaov1alpha1.OpenBaoPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if policy.DeletionTimestamp.IsZero() && policyDeletionPolicy(&policy).IsDelete() {
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
		return r.fail(ctx, &policy, "PolicyAcquireFailed", err)
	}
	if observed == nil {
		return r.fail(ctx, &policy, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no policy for %q", policy.Name))
	}
	if observed.Name != "" && observed.Name != policy.Name {
		return r.fail(ctx, &policy, "PolicyIdentityMismatch", fmt.Errorf("OpenBao policy is named %q, expected %q", observed.Name, policy.Name))
	}

	if observed.Rules != policy.Spec.Rules {
		if err := apiClient.WritePolicy(ctx, policy.Name, openbaoclient.PolicyRequest{Rules: policy.Spec.Rules}); err != nil {
			return r.fail(ctx, &policy, "PolicyWriteFailed", err)
		}
		observed, err = apiClient.GetPolicy(ctx, policy.Name)
		if err != nil {
			return r.fail(ctx, &policy, "PolicyReadFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &policy, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no policy after writing %q", policy.Name))
		}
	}

	observePolicyStatus(&policy, observed)
	markReady(&policy.Status.Conditions, policy.Generation, "OpenBao policy is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &policy, before, &policy.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: policyDriftInterval(&policy)}, nil
}

func (r *OpenBaoPolicyReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (PolicyClient, error) {
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

func (r *OpenBaoPolicyReconciler) acquire(ctx context.Context, policy *openbaov1alpha1.OpenBaoPolicy, apiClient PolicyClient) (*openbaoclient.Policy, error) {
	observed, err := apiClient.GetPolicy(ctx, policy.Name)
	if err == nil {
		if policy.Status.Name == "" && !policyCreationPolicy(policy).AllowsAdoption() {
			return nil, fmt.Errorf("OpenBao policy %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", policy.Name)
		}
		return observed, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	if !policyCreationPolicy(policy).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao policy %q does not exist and creationPolicy=%s does not allow creation", policy.Name, policyCreationPolicy(policy))
	}

	if err := apiClient.WritePolicy(ctx, policy.Name, openbaoclient.PolicyRequest{Rules: policy.Spec.Rules}); err != nil {
		return nil, err
	}
	return apiClient.GetPolicy(ctx, policy.Name)
}

func (r *OpenBaoPolicyReconciler) reconcileDeletion(ctx context.Context, policy *openbaov1alpha1.OpenBaoPolicy) (ctrl.Result, error) {
	if policyDeletionPolicy(policy) != openbaov1alpha1.DeletionPolicyDelete {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, policy)
	}
	connection, err := resolveConnection(ctx, r.Client, policy.Namespace, policy.Spec.ConnectionRef)
	if err != nil {
		if isNotFound(err) {
			return removeFinalizerAfterDependencyLoss(ctx, r.Client, policy, "OpenBaoConnection", err)
		}
		return ctrl.Result{}, err
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		if isNotFound(err) {
			return removeFinalizerAfterDependencyLoss(ctx, r.Client, policy, "OpenBaoConnection credentials", err)
		}
		return ctrl.Result{}, err
	}
	if err := apiClient.DeletePolicy(ctx, policy.Name); err != nil && !isNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, policy)
}

func (r *OpenBaoPolicyReconciler) dependencyFailure(ctx context.Context, policy *openbaov1alpha1.OpenBaoPolicy, err error) (ctrl.Result, error) {
	before := policy.Status.DeepCopy()
	policy.Status.ObservedGeneration = policy.Generation
	markStalled(&policy.Status.Conditions, policy.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, policy, before, &policy.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoPolicyReconciler) fail(ctx context.Context, policy *openbaov1alpha1.OpenBaoPolicy, reason string, err error) (ctrl.Result, error) {
	before := policy.Status.DeepCopy()
	policy.Status.ObservedGeneration = policy.Generation
	markError(&policy.Status.Conditions, policy.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, policy, before, &policy.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoPolicyReconciler) mapConnectionToPolicies(ctx context.Context, obj client.Object) []reconcile.Request {
	var policies openbaov1alpha1.OpenBaoPolicyList
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

// SetupWithManager sets up the policy controller and its connection watch.
func (r *OpenBaoPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoPolicy{}).
		Named("openbao-openbaopolicy").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToPolicies)).
		Complete(r)
}

func observePolicyStatus(policy *openbaov1alpha1.OpenBaoPolicy, observed *openbaoclient.Policy) {
	name := observed.Name
	if name == "" {
		name = policy.Name
	}
	policy.Status.Name = name
	policy.Status.RulesHash = policyRulesHash(observed.Rules)
	policy.Status.Version = observed.Version
	policy.Status.ObservedGeneration = policy.Generation
}

func policyRulesHash(rules string) string {
	digest := sha256.Sum256([]byte(rules))
	return hex.EncodeToString(digest[:])
}

func policyDriftInterval(policy *openbaov1alpha1.OpenBaoPolicy) time.Duration {
	if policy.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return policy.Spec.DriftDetectionInterval.Duration
}

func policyCreationPolicy(policy *openbaov1alpha1.OpenBaoPolicy) openbaov1alpha1.CreationPolicy {
	if policy.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return policy.Spec.CreationPolicy
}

func policyDeletionPolicy(policy *openbaov1alpha1.OpenBaoPolicy) openbaov1alpha1.DeletionPolicy {
	if policy.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return policy.Spec.DeletionPolicy
}
