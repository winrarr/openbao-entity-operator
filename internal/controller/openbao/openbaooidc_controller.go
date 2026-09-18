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

// OIDCClient is the OpenBao identity OIDC configuration surface.
type OIDCClient interface {
	GetOIDCResource(context.Context, string, string) (openbaoclient.OIDCObject, error)
	WriteOIDCResource(context.Context, string, string, openbaoclient.OIDCObject) error
	DeleteOIDCResource(context.Context, string, string) error
	GetOIDCConfig(context.Context) (openbaoclient.OIDCObject, error)
	WriteOIDCConfig(context.Context, openbaoclient.OIDCObject) error
}

type oidcStatusView struct {
	Name               *string
	ConfigHash         *string
	ObservedGeneration *int64
	Conditions         *[]metav1.Condition
	Before             any
	Current            any
}

type oidcNamedPlan struct {
	Object         client.Object
	ConnectionRef  openbaov1alpha1.OpenBaoConnectionReference
	CreationPolicy openbaov1alpha1.CreationPolicy
	DeletionPolicy openbaov1alpha1.DeletionPolicy
	DriftInterval  time.Duration
	Resource       string
	Name           string
	Desired        func() (openbaoclient.OIDCObject, error)
	Status         oidcStatusView
}

func reconcileOIDCNamed(ctx context.Context, kubeClient client.Client, clientCache *ConnectionClientCache, newClient func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error), plan oidcNamedPlan) (ctrl.Result, error) {
	if plan.Object.GetDeletionTimestamp().IsZero() && plan.DeletionPolicy.IsDelete() {
		if err := ensureFinalizer(ctx, kubeClient, plan.Object); err != nil {
			return ctrl.Result{}, err
		}
	}
	updateStatus := func() error {
		return updateStatusIfChanged(ctx, kubeClient, plan.Object, plan.Status.Before, plan.Status.Current)
	}
	if !plan.Object.GetDeletionTimestamp().IsZero() {
		if plan.DeletionPolicy != openbaov1alpha1.DeletionPolicyDelete {
			return ctrl.Result{}, removeFinalizer(ctx, kubeClient, plan.Object)
		}
		connection, err := resolveConnection(ctx, kubeClient, plan.Object.GetNamespace(), plan.ConnectionRef)
		if err != nil {
			return recordCleanupFailure(ctx, kubeClient, plan.Object, plan.Status.Before, plan.Status.Current, plan.Status.Conditions, plan.Object.GetGeneration(), reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, plan.Object, "OpenBaoConnection", err))
		}
		apiClient, err := oidcClientFor(ctx, kubeClient, clientCache, newClient, connection)
		if err != nil {
			return recordCleanupFailure(ctx, kubeClient, plan.Object, plan.Status.Before, plan.Status.Current, plan.Status.Conditions, plan.Object.GetGeneration(), reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, plan.Object, "OpenBaoConnection credentials", err))
		}
		if err := apiClient.DeleteOIDCResource(ctx, plan.Resource, plan.Name); err != nil && !isNotFound(err) {
			return recordCleanupFailure(ctx, kubeClient, plan.Object, plan.Status.Before, plan.Status.Current, plan.Status.Conditions, plan.Object.GetGeneration(), "CleanupFailed", err)
		}
		return ctrl.Result{}, removeFinalizer(ctx, kubeClient, plan.Object)
	}

	*plan.Status.ObservedGeneration = plan.Object.GetGeneration()
	connection, err := resolveConnection(ctx, kubeClient, plan.Object.GetNamespace(), plan.ConnectionRef)
	if err != nil {
		return oidcDependencyFailure(plan, updateStatus, fmt.Errorf("read OpenBaoConnection %s/%s: %w", plan.Object.GetNamespace(), plan.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return oidcDependencyFailure(plan, updateStatus, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}
	apiClient, err := oidcClientFor(ctx, kubeClient, clientCache, newClient, connection)
	if err != nil {
		return oidcDependencyFailure(plan, updateStatus, err)
	}

	observed, err := apiClient.GetOIDCResource(ctx, plan.Resource, plan.Name)
	if err != nil {
		if !isNotFound(err) {
			return oidcFailure(plan, updateStatus, "OIDCReadFailed", err)
		}
		if !plan.CreationPolicy.AllowsCreation() {
			return oidcFailure(plan, updateStatus, "OIDCAcquireFailed", fmt.Errorf("OpenBao OIDC %s %q does not exist and creationPolicy=%s does not allow creation", plan.Resource, plan.Name, plan.CreationPolicy))
		}
		desired, desiredErr := plan.Desired()
		if desiredErr != nil {
			return oidcFailure(plan, updateStatus, "InvalidOIDCSpec", desiredErr)
		}
		if err := apiClient.WriteOIDCResource(ctx, plan.Resource, plan.Name, desired); err != nil {
			return oidcFailure(plan, updateStatus, "OIDCWriteFailed", err)
		}
		observed, err = apiClient.GetOIDCResource(ctx, plan.Resource, plan.Name)
		if err != nil {
			return oidcFailure(plan, updateStatus, "OIDCReadFailed", err)
		}
	} else if *plan.Status.Name == "" && !plan.CreationPolicy.AllowsAdoption() {
		return oidcFailure(plan, updateStatus, "OIDCAcquireFailed", fmt.Errorf("OpenBao OIDC %s %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", plan.Resource, plan.Name))
	}
	if observed == nil {
		return oidcFailure(plan, updateStatus, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no OIDC %s %q", plan.Resource, plan.Name))
	}
	desired, err := plan.Desired()
	if err != nil {
		return oidcFailure(plan, updateStatus, "InvalidOIDCSpec", err)
	}
	if !oidcDesiredFieldsMatch(desired, observed) {
		if err := apiClient.WriteOIDCResource(ctx, plan.Resource, plan.Name, desired); err != nil {
			return oidcFailure(plan, updateStatus, "OIDCWriteFailed", err)
		}
		observed, err = apiClient.GetOIDCResource(ctx, plan.Resource, plan.Name)
		if err != nil {
			return oidcFailure(plan, updateStatus, "OIDCReadFailed", err)
		}
	}

	*plan.Status.Name = plan.Name
	*plan.Status.ConfigHash = oidcConfigHash(observed)
	markReady(plan.Status.Conditions, plan.Object.GetGeneration(), fmt.Sprintf("OpenBao OIDC %s is reconciled", plan.Resource))
	if err := updateStatus(); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: plan.DriftInterval}, nil
}

func oidcClientFor(ctx context.Context, kubeClient client.Client, cache *ConnectionClientCache, newClient func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error), connection *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error) {
	if newClient != nil {
		return newClient(ctx, connection)
	}
	if cache != nil {
		return cache.ClientFor(ctx, kubeClient, connection)
	}
	return connectionClientFor(ctx, kubeClient, connection)
}

func oidcDependencyFailure(plan oidcNamedPlan, update func() error, err error) (ctrl.Result, error) {
	markStalled(plan.Status.Conditions, plan.Object.GetGeneration(), "DependencyNotReady", err)
	if statusErr := update(); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func oidcFailure(plan oidcNamedPlan, update func() error, reason string, err error) (ctrl.Result, error) {
	markError(plan.Status.Conditions, plan.Object.GetGeneration(), reason, err)
	if statusErr := update(); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func oidcDesiredFieldsMatch(desired, observed openbaoclient.OIDCObject) bool {
	for key, desiredValue := range desired {
		observedValue, ok := observed[key]
		if !ok || !reflect.DeepEqual(canonicalOIDCValue(desiredValue), canonicalOIDCValue(observedValue)) {
			return false
		}
	}
	return true
}

func canonicalOIDCValue(value any) any {
	switch typed := value.(type) {
	case []string:
		result := append([]string(nil), typed...)
		slices.Sort(result)
		return result
	case []any:
		result := append([]any(nil), typed...)
		for i := range result {
			result[i] = canonicalOIDCValue(result[i])
		}
		slices.SortFunc(result, func(left, right any) int {
			leftJSON, _ := json.Marshal(left)
			rightJSON, _ := json.Marshal(right)
			return strings.Compare(string(leftJSON), string(rightJSON))
		})
		return result
	default:
		return value
	}
}

func oidcConfigHash(object openbaoclient.OIDCObject) string {
	data, err := json.Marshal(canonicalOIDCValue(object))
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func oidcDurationSeconds(value *metav1.Duration, field string) (int64, error) {
	return durationSeconds(value, field)
}

func oidcOptionalDuration(value *metav1.Duration, field string) (any, error) {
	if value == nil {
		return nil, nil
	}
	seconds, err := oidcDurationSeconds(value, field)
	if err != nil {
		return nil, err
	}
	return seconds, nil
}

func oidcStrings(values []string, field string) ([]string, error) {
	result := append([]string(nil), values...)
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
		if result[i] == "" {
			return nil, fmt.Errorf("%s must not contain empty values", field)
		}
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func oidcResourcePolicies(creation openbaov1alpha1.CreationPolicy, deletion openbaov1alpha1.DeletionPolicy) (openbaov1alpha1.CreationPolicy, openbaov1alpha1.DeletionPolicy) {
	if creation == "" {
		creation = openbaov1alpha1.CreationPolicyCreate
	}
	if deletion == "" {
		deletion = openbaov1alpha1.DeletionPolicyOrphan
	}
	return creation, deletion
}

// OpenBaoOIDCConfigReconciler reconciles the singleton OIDC issuer configuration.
type OpenBaoOIDCConfigReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *OpenBaoOIDCConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoOIDCConfig
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	object.Status.ObservedGeneration = object.Generation
	connection, err := resolveConnection(ctx, r.Client, object.Namespace, object.Spec.ConnectionRef)
	if err != nil {
		markStalled(&object.Status.Conditions, object.Generation, "DependencyNotReady", err)
		if statusErr := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: dependencyRetry}, nil
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		markStalled(&object.Status.Conditions, object.Generation, "DependencyNotReady", dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
		if statusErr := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: dependencyRetry}, nil
	}
	apiClient, err := oidcClientFor(ctx, r.Client, r.ClientCache, r.NewClient, connection)
	if err != nil {
		markError(&object.Status.Conditions, object.Generation, "OIDCClientFailed", err)
		if statusErr := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}
	desired := openbaoclient.OIDCObject{}
	if strings.TrimSpace(object.Spec.Issuer) != "" {
		desired["issuer"] = strings.TrimSpace(object.Spec.Issuer)
	}
	observed, err := apiClient.GetOIDCConfig(ctx)
	if err != nil {
		markError(&object.Status.Conditions, object.Generation, "OIDCReadFailed", err)
		if statusErr := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}
	if !oidcDesiredFieldsMatch(desired, observed) {
		if err := apiClient.WriteOIDCConfig(ctx, desired); err != nil {
			markError(&object.Status.Conditions, object.Generation, "OIDCWriteFailed", err)
			if statusErr := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, err
		}
		observed, err = apiClient.GetOIDCConfig(ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	object.Status.ConfigHash = oidcConfigHash(observed)
	markReady(&object.Status.Conditions, object.Generation, "OpenBao OIDC configuration is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: defaultDriftCheck}, nil
}

func (r *OpenBaoOIDCConfigReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var objects openbaov1alpha1.OpenBaoOIDCConfigList
	if err := r.List(ctx, &objects, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range objects.Items {
		if objects.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&objects.Items[i])})
		}
	}
	return requests
}

func (r *OpenBaoOIDCConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCConfig{}).Named("openbao-openbaooidcconfig").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

// The remaining OIDC reconcilers are thin typed adapters over the shared
// lifecycle implementation above.
type OpenBaoOIDCProviderReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

type OpenBaoOIDCClientReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

type OpenBaoOIDCKeyReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

type OpenBaoOIDCRoleReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

type OpenBaoOIDCScopeReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

type OpenBaoOIDCAssignmentReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcproviders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcproviders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcproviders/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao OIDC provider.
func (r *OpenBaoOIDCProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoOIDCProvider
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	creation, deletion := oidcResourcePolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	before := o.Status.DeepCopy()
	return reconcileOIDCNamed(ctx, r.Client, r.ClientCache, r.NewClient, oidcNamedPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: oidcDrift(o.Spec.DriftDetectionInterval), Resource: "provider", Name: o.Name, Desired: func() (openbaoclient.OIDCObject, error) {
		result := openbaoclient.OIDCObject{}
		if strings.TrimSpace(o.Spec.Issuer) != "" {
			result["issuer"] = strings.TrimSpace(o.Spec.Issuer)
		}
		if len(o.Spec.AllowedClientIDs) > 0 {
			values, err := oidcStrings(o.Spec.AllowedClientIDs, "allowedClientIDs")
			if err != nil {
				return nil, err
			}
			result["allowed_client_ids"] = values
		}
		if len(o.Spec.ScopesSupported) > 0 {
			values, err := oidcStrings(o.Spec.ScopesSupported, "scopesSupported")
			if err != nil {
				return nil, err
			}
			result["scopes_supported"] = values
		}
		return result, nil
	}, Status: oidcStatusView{Name: &o.Status.Name, ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}})
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcclients/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao OIDC client.
func (r *OpenBaoOIDCClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoOIDCClient
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	creation, deletion := oidcResourcePolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	before := o.Status.DeepCopy()
	desired := func() (openbaoclient.OIDCObject, error) {
		result := openbaoclient.OIDCObject{}
		values, err := oidcOptionalDuration(o.Spec.AccessTokenTTL, "accessTokenTTL")
		if err != nil {
			return nil, err
		}
		if values != nil {
			result["access_token_ttl"] = values
		}
		if len(o.Spec.Assignments) > 0 {
			result["assignments"], err = oidcStrings(o.Spec.Assignments, "assignments")
			if err != nil {
				return nil, err
			}
		}
		if o.Spec.AuthorizationCode != nil {
			result["authorization_code"] = *o.Spec.AuthorizationCode
		}
		if o.Spec.ClientCredentials != nil {
			result["client_credentials"] = *o.Spec.ClientCredentials
		}
		if strings.TrimSpace(o.Spec.ClientType) != "" {
			result["client_type"] = strings.TrimSpace(o.Spec.ClientType)
		}
		values, err = oidcOptionalDuration(o.Spec.IDTokenTTL, "idTokenTTL")
		if err != nil {
			return nil, err
		}
		if values != nil {
			result["id_token_ttl"] = values
		}
		if strings.TrimSpace(o.Spec.Key) != "" {
			result["key"] = strings.TrimSpace(o.Spec.Key)
		}
		if len(o.Spec.RedirectURIs) > 0 {
			result["redirect_uris"], err = oidcStrings(o.Spec.RedirectURIs, "redirectURIs")
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	return reconcileOIDCNamed(ctx, r.Client, r.ClientCache, r.NewClient, oidcNamedPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: oidcDrift(o.Spec.DriftDetectionInterval), Resource: "client", Name: o.Name, Desired: desired, Status: oidcStatusView{Name: &o.Status.Name, ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}})
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidckeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidckeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidckeys/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao OIDC key.
func (r *OpenBaoOIDCKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoOIDCKey
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	creation, deletion := oidcResourcePolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	before := o.Status.DeepCopy()
	return reconcileOIDCNamed(ctx, r.Client, r.ClientCache, r.NewClient, oidcNamedPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: oidcDrift(o.Spec.DriftDetectionInterval), Resource: "key", Name: o.Name, Desired: func() (openbaoclient.OIDCObject, error) {
		result := openbaoclient.OIDCObject{}
		if strings.TrimSpace(o.Spec.Algorithm) != "" {
			result["algorithm"] = strings.TrimSpace(o.Spec.Algorithm)
		}
		if len(o.Spec.AllowedClientIDs) > 0 {
			values, err := oidcStrings(o.Spec.AllowedClientIDs, "allowedClientIDs")
			if err != nil {
				return nil, err
			}
			result["allowed_client_ids"] = values
		}
		values, err := oidcOptionalDuration(o.Spec.RotationPeriod, "rotationPeriod")
		if err != nil {
			return nil, err
		}
		if values != nil {
			result["rotation_period"] = values
		}
		values, err = oidcOptionalDuration(o.Spec.VerificationTTL, "verificationTTL")
		if err != nil {
			return nil, err
		}
		if values != nil {
			result["verification_ttl"] = values
		}
		return result, nil
	}, Status: oidcStatusView{Name: &o.Status.Name, ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}})
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcroles/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao OIDC role.
func (r *OpenBaoOIDCRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoOIDCRole
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	creation, deletion := oidcResourcePolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	before := o.Status.DeepCopy()
	return reconcileOIDCNamed(ctx, r.Client, r.ClientCache, r.NewClient, oidcNamedPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: oidcDrift(o.Spec.DriftDetectionInterval), Resource: "role", Name: o.Name, Desired: func() (openbaoclient.OIDCObject, error) {
		result := openbaoclient.OIDCObject{"key": strings.TrimSpace(o.Spec.Key)}
		if result["key"] == "" {
			return nil, fmt.Errorf("key must not be empty")
		}
		if strings.TrimSpace(o.Spec.ClientID) != "" {
			result["client_id"] = strings.TrimSpace(o.Spec.ClientID)
		}
		if o.Spec.Template != "" {
			result["template"] = o.Spec.Template
		}
		value, err := oidcOptionalDuration(o.Spec.TTL, "ttl")
		if err != nil {
			return nil, err
		}
		if value != nil {
			result["ttl"] = value
		}
		return result, nil
	}, Status: oidcStatusView{Name: &o.Status.Name, ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}})
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcscopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcscopes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcscopes/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao OIDC scope.
func (r *OpenBaoOIDCScopeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoOIDCScope
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	creation, deletion := oidcResourcePolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	before := o.Status.DeepCopy()
	return reconcileOIDCNamed(ctx, r.Client, r.ClientCache, r.NewClient, oidcNamedPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: oidcDrift(o.Spec.DriftDetectionInterval), Resource: "scope", Name: o.Name, Desired: func() (openbaoclient.OIDCObject, error) {
		result := openbaoclient.OIDCObject{}
		if o.Spec.Description != "" {
			result["description"] = o.Spec.Description
		}
		if o.Spec.Template != "" {
			result["template"] = o.Spec.Template
		}
		return result, nil
	}, Status: oidcStatusView{Name: &o.Status.Name, ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}})
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcassignments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaooidcassignments/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao OIDC assignment.
func (r *OpenBaoOIDCAssignmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoOIDCAssignment
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	creation, deletion := oidcResourcePolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	before := o.Status.DeepCopy()
	return reconcileOIDCNamed(ctx, r.Client, r.ClientCache, r.NewClient, oidcNamedPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: oidcDrift(o.Spec.DriftDetectionInterval), Resource: "assignment", Name: o.Name, Desired: func() (openbaoclient.OIDCObject, error) {
		result := openbaoclient.OIDCObject{}
		var err error
		if len(o.Spec.EntityIDs) > 0 {
			result["entity_ids"], err = oidcStrings(o.Spec.EntityIDs, "entityIDs")
			if err != nil {
				return nil, err
			}
		}
		if len(o.Spec.GroupIDs) > 0 {
			result["group_ids"], err = oidcStrings(o.Spec.GroupIDs, "groupIDs")
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}, Status: oidcStatusView{Name: &o.Status.Name, ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}})
}

func oidcDrift(duration *metav1.Duration) time.Duration {
	if duration == nil {
		return defaultDriftCheck
	}
	return duration.Duration
}

func (r *OpenBaoOIDCProviderReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoOIDCProviderList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}
func (r *OpenBaoOIDCClientReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoOIDCClientList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}
func (r *OpenBaoOIDCKeyReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoOIDCKeyList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}
func (r *OpenBaoOIDCRoleReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoOIDCRoleList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}
func (r *OpenBaoOIDCScopeReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoOIDCScopeList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}
func (r *OpenBaoOIDCAssignmentReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoOIDCAssignmentList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}

func (r *OpenBaoOIDCProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCProvider{}).Named("openbao-openbaooidcprovider").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
func (r *OpenBaoOIDCClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCClient{}).Named("openbao-openbaooidcclient").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
func (r *OpenBaoOIDCKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCKey{}).Named("openbao-openbaooidckey").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
func (r *OpenBaoOIDCRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCRole{}).Named("openbao-openbaooidcrole").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
func (r *OpenBaoOIDCScopeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCScope{}).Named("openbao-openbaooidcscope").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
func (r *OpenBaoOIDCAssignmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoOIDCAssignment{}).Named("openbao-openbaooidcassignment").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
