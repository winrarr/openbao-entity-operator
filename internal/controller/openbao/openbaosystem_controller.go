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
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

// SystemClient is the declarative OpenBao system-configuration surface.
type SystemClient interface {
	GetMFALoginEnforcement(context.Context, string) (openbaoclient.AdditionalObject, error)
	WriteMFALoginEnforcement(context.Context, string, openbaoclient.AdditionalObject) error
	DeleteMFALoginEnforcement(context.Context, string) error
	GetCORSConfiguration(context.Context) (openbaoclient.AdditionalObject, error)
	WriteCORSConfiguration(context.Context, openbaoclient.AdditionalObject) error
	DeleteCORSConfiguration(context.Context) error
	GetAuditRequestHeader(context.Context, string) (openbaoclient.AdditionalObject, error)
	WriteAuditRequestHeader(context.Context, string, openbaoclient.AdditionalObject) error
	DeleteAuditRequestHeader(context.Context, string) error
	GetUIHeader(context.Context, string) (openbaoclient.AdditionalObject, error)
	WriteUIHeader(context.Context, string, openbaoclient.AdditionalObject) error
	DeleteUIHeader(context.Context, string) error
	GetRateLimitQuotaConfiguration(context.Context) (openbaoclient.AdditionalObject, error)
	WriteRateLimitQuotaConfiguration(context.Context, openbaoclient.AdditionalObject) error
	GetLogger(context.Context, string) (openbaoclient.AdditionalObject, error)
	WriteLogger(context.Context, string, openbaoclient.AdditionalObject) error
	DeleteLogger(context.Context, string) error
	GetEncryptionKeyConfiguration(context.Context) (openbaoclient.AdditionalObject, error)
	WriteEncryptionKeyConfiguration(context.Context, openbaoclient.AdditionalObject) error
	GetKeyringRotationConfiguration(context.Context) (openbaoclient.AdditionalObject, error)
	WriteKeyringRotationConfiguration(context.Context, openbaoclient.AdditionalObject) error
	GetAuthMethod(context.Context, string) (*openbaoclient.Mount, error)
	EnableAuthMethod(context.Context, string, map[string]any) error
	TuneAuthMethod(context.Context, string, map[string]any) error
	DisableAuthMethod(context.Context, string) error
	GetSecretEngine(context.Context, string) (*openbaoclient.Mount, error)
	EnableSecretEngine(context.Context, string, map[string]any) error
	TuneSecretEngine(context.Context, string, map[string]any) error
	DisableSecretEngine(context.Context, string) error
	GetNamespace(context.Context, string) (*openbaoclient.Namespace, error)
	WriteNamespace(context.Context, string, map[string]any) error
	DeleteNamespace(context.Context, string) error
	GetAuditDevice(context.Context, string) (*openbaoclient.AuditDevice, error)
	WriteAuditDevice(context.Context, string, map[string]any) error
	DeleteAuditDevice(context.Context, string) error
	GetRateLimitQuota(context.Context, string) (openbaoclient.RateLimitQuota, error)
	WriteRateLimitQuota(context.Context, string, openbaoclient.RateLimitQuota) error
	DeleteRateLimitQuota(context.Context, string) error
	GetWorkflow(context.Context, string) (openbaoclient.Workflow, error)
	WriteWorkflow(context.Context, string, openbaoclient.Workflow) error
	DeleteWorkflow(context.Context, string) error
	GetPlugin(context.Context, string, string) (openbaoclient.Plugin, error)
	WritePlugin(context.Context, string, string, openbaoclient.Plugin) error
	DeletePlugin(context.Context, string, string) error
}

const systemTypeField = "type"

type systemPlan struct {
	Object         client.Object
	ConnectionRef  openbaov1alpha1.OpenBaoConnectionReference
	CreationPolicy openbaov1alpha1.CreationPolicy
	DeletionPolicy openbaov1alpha1.DeletionPolicy
	DriftInterval  time.Duration
	Name           string
	Desired        func() (map[string]any, error)
	Get            func(SystemClient) (map[string]any, error)
	Write          func(SystemClient, map[string]any, bool) error
	Delete         func(SystemClient) error
	Acquired       func() bool
	MarkAcquired   func()
	Observe        func(map[string]any)
	Status         oidcStatusView
}

func reconcileSystemNamed(ctx context.Context, kubeClient client.Client, cache *ConnectionClientCache, newClient func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error), plan systemPlan) (ctrl.Result, error) {
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
		apiClient, err := systemClientFor(ctx, kubeClient, cache, newClient, connection)
		if err != nil {
			return recordCleanupFailure(ctx, kubeClient, plan.Object, plan.Status.Before, plan.Status.Current, plan.Status.Conditions, plan.Object.GetGeneration(), reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, plan.Object, "OpenBaoConnection credentials", err))
		}
		if err := plan.Delete(apiClient); err != nil && !isNotFound(err) {
			return recordCleanupFailure(ctx, kubeClient, plan.Object, plan.Status.Before, plan.Status.Current, plan.Status.Conditions, plan.Object.GetGeneration(), "CleanupFailed", err)
		}
		return ctrl.Result{}, removeFinalizer(ctx, kubeClient, plan.Object)
	}

	*plan.Status.ObservedGeneration = plan.Object.GetGeneration()
	connection, err := resolveConnection(ctx, kubeClient, plan.Object.GetNamespace(), plan.ConnectionRef)
	if err != nil {
		return systemDependencyFailure(plan, updateStatus, fmt.Errorf("read OpenBaoConnection %s/%s: %w", plan.Object.GetNamespace(), plan.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return systemDependencyFailure(plan, updateStatus, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}
	apiClient, err := systemClientFor(ctx, kubeClient, cache, newClient, connection)
	if err != nil {
		return systemDependencyFailure(plan, updateStatus, err)
	}

	observed, err := plan.Get(apiClient)
	created := false
	if err != nil {
		if !isNotFound(err) {
			return systemFailure(plan, updateStatus, "OpenBaoReadFailed", err)
		}
		if !plan.CreationPolicy.AllowsCreation() {
			return systemFailure(plan, updateStatus, "OpenBaoAcquireFailed", fmt.Errorf("OpenBao resource %q does not exist and creationPolicy=%s does not allow creation", plan.Name, plan.CreationPolicy))
		}
		created = true
	} else if !plan.Acquired() && !plan.CreationPolicy.AllowsAdoption() {
		return systemFailure(plan, updateStatus, "OpenBaoAcquireFailed", fmt.Errorf("OpenBao resource %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", plan.Name))
	}

	desired, err := plan.Desired()
	if err != nil {
		return systemFailure(plan, updateStatus, "InvalidOpenBaoSpec", err)
	}
	if created || observed == nil || !systemDesiredFieldsMatch(desired, observed) {
		if err := plan.Write(apiClient, desired, created); err != nil {
			return systemFailure(plan, updateStatus, "OpenBaoWriteFailed", err)
		}
		observed, err = plan.Get(apiClient)
		if err != nil {
			return systemFailure(plan, updateStatus, "OpenBaoReadFailed", err)
		}
	}
	if observed == nil {
		return systemFailure(plan, updateStatus, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no resource %q", plan.Name))
	}
	plan.MarkAcquired()
	plan.Observe(observed)
	*plan.Status.ConfigHash = systemConfigHash(observed)
	markReady(plan.Status.Conditions, plan.Object.GetGeneration(), "OpenBao resource is reconciled")
	if err := updateStatus(); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: plan.DriftInterval}, nil
}

func systemClientFor(ctx context.Context, kubeClient client.Client, cache *ConnectionClientCache, newClient func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error), connection *openbaov1alpha1.OpenBaoConnection) (SystemClient, error) {
	if newClient != nil {
		return newClient(ctx, connection)
	}
	if cache != nil {
		return cache.ClientFor(ctx, kubeClient, connection)
	}
	return connectionClientFor(ctx, kubeClient, connection)
}

func systemDependencyFailure(plan systemPlan, update func() error, err error) (ctrl.Result, error) {
	markStalled(plan.Status.Conditions, plan.Object.GetGeneration(), "DependencyNotReady", err)
	if statusErr := update(); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}
func systemFailure(plan systemPlan, update func() error, reason string, err error) (ctrl.Result, error) {
	markError(plan.Status.Conditions, plan.Object.GetGeneration(), reason, err)
	if statusErr := update(); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func systemDesiredFieldsMatch(desired, observed map[string]any) bool {
	for key, desiredValue := range desired {
		observedValue, ok := observed[key]
		if !ok || !reflectJSONEqual(desiredValue, observedValue) {
			return false
		}
	}
	return true
}
func reflectJSONEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}
func systemConfigHash(object map[string]any) string {
	data, err := json.Marshal(object)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
func objectMap(value any) map[string]any {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var result map[string]any
	if json.Unmarshal(data, &result) != nil {
		return nil
	}
	return result
}
func systemDuration(value *metav1.Duration, field string) (any, error) {
	if value == nil {
		return nil, nil
	}
	seconds, err := durationSeconds(value, field)
	if err != nil {
		return nil, err
	}
	return seconds, nil
}
func systemPolicies(creation openbaov1alpha1.CreationPolicy, deletion openbaov1alpha1.DeletionPolicy) (openbaov1alpha1.CreationPolicy, openbaov1alpha1.DeletionPolicy) {
	if creation == "" {
		creation = openbaov1alpha1.CreationPolicyCreate
	}
	if deletion == "" {
		deletion = openbaov1alpha1.DeletionPolicyOrphan
	}
	return creation, deletion
}

func mountDesired(spec openbaov1alpha1.OpenBaoMountSpec) (map[string]any, error) {
	result := map[string]any{systemTypeField: strings.TrimSpace(spec.Type)}
	if result[systemTypeField] == "" {
		return nil, fmt.Errorf("type must not be empty")
	}
	if spec.Description != "" {
		result["description"] = spec.Description
	}
	if spec.PluginName != "" {
		result["plugin_name"] = spec.PluginName
	}
	if spec.PluginVersion != "" {
		result["plugin_version"] = spec.PluginVersion
	}
	if spec.Local != nil {
		result["local"] = *spec.Local
	}
	if spec.SealWrap != nil {
		result["seal_wrap"] = *spec.SealWrap
	}
	if spec.ExternalEntropyAccess != nil {
		result["external_entropy_access"] = *spec.ExternalEntropyAccess
	}
	if spec.Options != nil {
		result["options"] = spec.Options
	}
	if spec.Config != nil {
		result["config"] = spec.Config
	}
	return result, nil
}
func mountTune(spec openbaov1alpha1.OpenBaoMountSpec) (map[string]any, error) {
	result := map[string]any{}
	if spec.Description != "" {
		result["description"] = spec.Description
	}
	if value, err := systemDuration(spec.DefaultLeaseTTL, "defaultLeaseTTL"); err != nil {
		return nil, err
	} else if value != nil {
		result["default_lease_ttl"] = fmt.Sprintf("%ds", value)
	}
	if value, err := systemDuration(spec.MaxLeaseTTL, "maxLeaseTTL"); err != nil {
		return nil, err
	} else if value != nil {
		result["max_lease_ttl"] = fmt.Sprintf("%ds", value)
	}
	if spec.ListingVisibility != "" {
		result["listing_visibility"] = spec.ListingVisibility
	}
	if spec.TokenType != "" {
		result["token_type"] = spec.TokenType
	}
	if len(spec.PassthroughRequestHeaders) > 0 {
		result["passthrough_request_headers"] = spec.PassthroughRequestHeaders
	}
	if len(spec.AllowedResponseHeaders) > 0 {
		result["allowed_response_headers"] = spec.AllowedResponseHeaders
	}
	return result, nil
}

// OpenBaoAuthMethodReconciler reconciles auth-method mounts and tuning.
type OpenBaoAuthMethodReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauthmethods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauthmethods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauthmethods/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao auth-method mount.
func (r *OpenBaoAuthMethodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoAuthMethod
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := o.Status.DeepCopy()
	return r.reconcileAuth(ctx, &o, before)
}
func (r *OpenBaoAuthMethodReconciler) reconcileAuth(ctx context.Context, o *openbaov1alpha1.OpenBaoAuthMethod, before *openbaov1alpha1.OpenBaoAuthMethodStatus) (ctrl.Result, error) {
	creation, deletion := systemPolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	plan := systemPlan{Object: o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(o.Spec.DriftDetectionInterval), Name: o.Spec.Path, Desired: func() (map[string]any, error) { return mountDesired(o.Spec.OpenBaoMountSpec) }, Get: func(c SystemClient) (map[string]any, error) {
		value, err := c.GetAuthMethod(ctx, o.Spec.Path)
		if err != nil {
			return nil, err
		}
		return objectMap(value), nil
	}, Write: func(c SystemClient, value map[string]any, creating bool) error {
		if creating {
			if err := c.EnableAuthMethod(ctx, o.Spec.Path, value); err != nil {
				return err
			}
		}
		tune, err := mountTune(o.Spec.OpenBaoMountSpec)
		if err != nil {
			return err
		}
		if len(tune) > 0 {
			return c.TuneAuthMethod(ctx, o.Spec.Path, tune)
		}
		return nil
	}, Delete: func(c SystemClient) error { return c.DisableAuthMethod(ctx, o.Spec.Path) }, Acquired: func() bool { return o.Status.Path != "" }, MarkAcquired: func() { o.Status.Path = o.Spec.Path }, Observe: func(value map[string]any) {
		if accessor, ok := value["accessor"].(string); ok {
			o.Status.Accessor = accessor
		}
		if typ, ok := value[systemTypeField].(string); ok {
			o.Status.Type = typ
		}
		o.Status.Path = o.Spec.Path
	}, Status: oidcStatusView{ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

// OpenBaoSecretEngineReconciler reconciles secret-engine mounts and tuning.
type OpenBaoSecretEngineReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaosecretengines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaosecretengines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaosecretengines/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao secret-engine mount.
func (r *OpenBaoSecretEngineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var o openbaov1alpha1.OpenBaoSecretEngine
	if err := r.Get(ctx, req.NamespacedName, &o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := o.Status.DeepCopy()
	creation, deletion := systemPolicies(o.Spec.CreationPolicy, o.Spec.DeletionPolicy)
	plan := systemPlan{Object: &o, ConnectionRef: o.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(o.Spec.DriftDetectionInterval), Name: o.Spec.Path, Desired: func() (map[string]any, error) { return mountDesired(o.Spec.OpenBaoMountSpec) }, Get: func(c SystemClient) (map[string]any, error) {
		value, err := c.GetSecretEngine(ctx, o.Spec.Path)
		if err != nil {
			return nil, err
		}
		return objectMap(value), nil
	}, Write: func(c SystemClient, value map[string]any, creating bool) error {
		if creating {
			if err := c.EnableSecretEngine(ctx, o.Spec.Path, value); err != nil {
				return err
			}
		}
		tune, err := mountTune(o.Spec.OpenBaoMountSpec)
		if err != nil {
			return err
		}
		if len(tune) > 0 {
			return c.TuneSecretEngine(ctx, o.Spec.Path, tune)
		}
		return nil
	}, Delete: func(c SystemClient) error { return c.DisableSecretEngine(ctx, o.Spec.Path) }, Acquired: func() bool { return o.Status.Path != "" }, MarkAcquired: func() { o.Status.Path = o.Spec.Path }, Observe: func(value map[string]any) {
		if accessor, ok := value["accessor"].(string); ok {
			o.Status.Accessor = accessor
		}
		if typ, ok := value[systemTypeField].(string); ok {
			o.Status.Type = typ
		}
		o.Status.Path = o.Spec.Path
	}, Status: oidcStatusView{ConfigHash: &o.Status.ConfigHash, ObservedGeneration: &o.Status.ObservedGeneration, Conditions: &o.Status.Conditions, Before: before, Current: &o.Status}}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func systemDrift(value *metav1.Duration) time.Duration {
	if value == nil {
		return defaultDriftCheck
	}
	return value.Duration
}
