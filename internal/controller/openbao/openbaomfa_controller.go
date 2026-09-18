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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
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

type OpenBaoMFALoginEnforcementReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaomfaloginenforcements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaomfaloginenforcements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaomfaloginenforcements/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao MFA login enforcement.
func (r *OpenBaoMFALoginEnforcementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoMFALoginEnforcement
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := mfaPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{
		Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion,
		DriftInterval: mfaDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Name,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Name) == "" {
				return nil, fmt.Errorf("name must not be empty")
			}
			methodIDs, err := mfaStrings(object.Spec.MFAMethodIDs, "mfaMethodIDs")
			if err != nil || len(methodIDs) == 0 {
				if err == nil {
					err = fmt.Errorf("mfaMethodIDs must contain at least one method ID")
				}
				return nil, err
			}
			result := openbaoclient.AdditionalObject{"mfa_method_ids": methodIDs}
			for key, values := range map[string][]string{
				"auth_method_accessors": object.Spec.AuthMethodAccessors,
				"auth_method_types":     object.Spec.AuthMethodTypes,
				"identity_entity_ids":   object.Spec.IdentityEntityIDs,
				"identity_group_ids":    object.Spec.IdentityGroupIDs,
			} {
				if len(values) > 0 {
					cleaned, err := mfaStrings(values, key)
					if err != nil {
						return nil, err
					}
					result[key] = cleaned
				}
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) {
			value, err := c.GetMFALoginEnforcement(ctx, object.Spec.Name)
			return value, err
		},
		Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteMFALoginEnforcement(ctx, object.Spec.Name, value)
		},
		Delete:       func(c SystemClient) error { return c.DeleteMFALoginEnforcement(ctx, object.Spec.Name) },
		Acquired:     func() bool { return object.Status.Name != "" },
		MarkAcquired: func() { object.Status.Name = object.Spec.Name },
		Observe:      func(_ map[string]any) { object.Status.Name = object.Spec.Name },
		Status:       oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoMFALoginEnforcementReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoMFALoginEnforcementList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}

// SetupWithManager sets up the MFA login-enforcement controller.
func (r *OpenBaoMFALoginEnforcementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoMFALoginEnforcement{}).Named("openbao-openbaomfaloginenforcement").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

type MFAMethodClient interface {
	GetMFAMethod(context.Context, string, string) (openbaoclient.AdditionalObject, error)
	WriteMFAMethod(context.Context, string, string, openbaoclient.AdditionalObject) error
	DeleteMFAMethod(context.Context, string, string) error
}

type OpenBaoMFAMethodReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (MFAMethodClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaomfamethods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaomfamethods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaomfamethods/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao MFA method.
func (r *OpenBaoMFAMethodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoMFAMethod
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if object.DeletionTimestamp.IsZero() && object.Spec.DeletionPolicy.IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &object); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !object.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &object)
	}
	before := object.Status.DeepCopy()
	object.Status.ObservedGeneration = object.Generation
	connection, err := resolveConnection(ctx, r.Client, object.Namespace, object.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &object, err)
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &object, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &object, err)
	}
	desired, secretHash, err := r.desired(ctx, &object)
	if err != nil {
		return r.fail(ctx, &object, "InvalidMFASpec", err)
	}
	observed, err := apiClient.GetMFAMethod(ctx, string(object.Spec.Type), object.Spec.MethodID)
	created := false
	if err != nil {
		if !isNotFound(err) {
			return r.fail(ctx, &object, "MFAReadFailed", err)
		}
		if !mfaCreation(&object).AllowsCreation() {
			return r.fail(ctx, &object, "MFIAcquireFailed", fmt.Errorf("OpenBao MFA method %q does not exist and creationPolicy=%s does not allow creation", object.Spec.MethodID, mfaCreation(&object)))
		}
		created = true
	} else if object.Status.MethodID == "" && !mfaCreation(&object).AllowsAdoption() {
		return r.fail(ctx, &object, "MFIAcquireFailed", fmt.Errorf("OpenBao MFA method %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", object.Spec.MethodID))
	}
	if created || object.Status.SecretHash != secretHash || !mfaFieldsMatch(desired, observed) {
		if err := apiClient.WriteMFAMethod(ctx, string(object.Spec.Type), object.Spec.MethodID, desired); err != nil {
			return r.fail(ctx, &object, "MFAWriteFailed", err)
		}
		observed, err = apiClient.GetMFAMethod(ctx, string(object.Spec.Type), object.Spec.MethodID)
		if err != nil {
			return r.fail(ctx, &object, "MFAReadFailed", err)
		}
	}
	object.Status.MethodID = object.Spec.MethodID
	object.Status.MethodName = object.Spec.MethodName
	if value, ok := observed["method_name"].(string); ok && value != "" {
		object.Status.MethodName = value
	}
	object.Status.ConfigHash = mfaHash(observed)
	object.Status.SecretHash = secretHash
	markReady(&object.Status.Conditions, object.Generation, "OpenBao MFA method is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &object, before, &object.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: mfaDrift(object.Spec.DriftDetectionInterval)}, nil
}

func (r *OpenBaoMFAMethodReconciler) desired(ctx context.Context, object *openbaov1alpha1.OpenBaoMFAMethod) (openbaoclient.AdditionalObject, string, error) {
	if strings.TrimSpace(object.Spec.MethodID) == "" || strings.TrimSpace(object.Spec.MethodName) == "" {
		return nil, "", fmt.Errorf("methodID and methodName must not be empty")
	}
	result := openbaoclient.AdditionalObject{"method_name": object.Spec.MethodName}
	if object.Spec.UsernameFormat != "" {
		result["username_format"] = object.Spec.UsernameFormat
	}
	secretParts := make([]string, 0, 3)
	addString := func(key, value string) {
		if value != "" {
			result[key] = value
		}
	}
	switch object.Spec.Type {
	case openbaov1alpha1.OpenBaoMFAMethodTypeDuo:
		addString("api_hostname", object.Spec.APIHostname)
		addString("integration_key", object.Spec.IntegrationKey)
		if object.Spec.SecretKeyRef == nil {
			return nil, "", fmt.Errorf("secretKeyRef is required for duo MFA")
		}
		value, err := r.readSecret(ctx, object.Namespace, *object.Spec.SecretKeyRef, "value")
		if err != nil {
			return nil, "", err
		}
		result["secret_key"] = value
		secretParts = append(secretParts, "duo="+value)
		addString("push_info", object.Spec.PushInfo)
		if object.Spec.UsePasscode != nil {
			result["use_passcode"] = *object.Spec.UsePasscode
		}
	case openbaov1alpha1.OpenBaoMFAMethodTypeOkta:
		if object.Spec.APITokenRef == nil {
			return nil, "", fmt.Errorf("apiTokenRef is required for okta MFA")
		}
		value, err := r.readSecret(ctx, object.Namespace, *object.Spec.APITokenRef, "value")
		if err != nil {
			return nil, "", err
		}
		result["api_token"] = value
		secretParts = append(secretParts, "okta="+value)
		addString("base_url", object.Spec.BaseURL)
		addString("org_name", object.Spec.OrgName)
		if object.Spec.PrimaryEmail != nil {
			result["primary_email"] = *object.Spec.PrimaryEmail
		}
		if object.Spec.Production != nil {
			result["production"] = *object.Spec.Production
		}
	case openbaov1alpha1.OpenBaoMFAMethodTypePingID:
		if object.Spec.SettingsFileRef == nil {
			return nil, "", fmt.Errorf("settingsFileRef is required for pingid MFA")
		}
		value, err := r.readSecret(ctx, object.Namespace, *object.Spec.SettingsFileRef, "value")
		if err != nil {
			return nil, "", err
		}
		result["settings_file_base64"] = base64.StdEncoding.EncodeToString([]byte(value))
		secretParts = append(secretParts, "pingid="+value)
	case openbaov1alpha1.OpenBaoMFAMethodTypeTOTP:
		addString("algorithm", object.Spec.Algorithm)
		if object.Spec.Digits != nil {
			result["digits"] = *object.Spec.Digits
		}
		addString("issuer", object.Spec.Issuer)
		if object.Spec.KeySize != nil {
			result["key_size"] = *object.Spec.KeySize
		}
		if object.Spec.MaxValidationAttempts != nil {
			result["max_validation_attempts"] = *object.Spec.MaxValidationAttempts
		}
		if object.Spec.Period != nil {
			result["period"] = *object.Spec.Period
		}
		if object.Spec.QRSize != nil {
			result["qr_size"] = *object.Spec.QRSize
		}
		if object.Spec.Skew != nil {
			result["skew"] = *object.Spec.Skew
		}
	default:
		return nil, "", fmt.Errorf("unsupported MFA type %q", object.Spec.Type)
	}
	return result, mfaSecretHash(secretParts), nil
}

func (r *OpenBaoMFAMethodReconciler) readSecret(ctx context.Context, namespace string, ref openbaov1alpha1.SecretKeyReference, defaultKey string) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
		return "", fmt.Errorf("read MFA Secret %s/%s: %w", namespace, ref.Name, err)
	}
	key := ref.Key
	if key == "" {
		key = defaultKey
	}
	value, ok := secret.Data[key]
	if !ok || len(value) == 0 {
		return "", fmt.Errorf("MFA Secret %s/%s does not contain key %q", namespace, ref.Name, key)
	}
	return string(value), nil
}

func (r *OpenBaoMFAMethodReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (MFAMethodClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoMFAMethodReconciler) reconcileDeletion(ctx context.Context, object *openbaov1alpha1.OpenBaoMFAMethod) (ctrl.Result, error) {
	if object.Spec.DeletionPolicy != openbaov1alpha1.DeletionPolicyDelete || object.Status.MethodID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, object)
	}
	before := object.Status.DeepCopy()
	connection, err := resolveConnection(ctx, r.Client, object.Namespace, object.Spec.ConnectionRef)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, object, before, &object.Status, &object.Status.Conditions, object.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, object, "OpenBaoConnection", err))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, object, before, &object.Status, &object.Status.Conditions, object.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, object, "OpenBaoConnection credentials", err))
	}
	if err := apiClient.DeleteMFAMethod(ctx, string(object.Spec.Type), object.Status.MethodID); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, object, before, &object.Status, &object.Status.Conditions, object.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, object)
}

func (r *OpenBaoMFAMethodReconciler) dependencyFailure(ctx context.Context, object *openbaov1alpha1.OpenBaoMFAMethod, err error) (ctrl.Result, error) {
	before := object.Status.DeepCopy()
	markStalled(&object.Status.Conditions, object.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, object, before, &object.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoMFAMethodReconciler) fail(ctx context.Context, object *openbaov1alpha1.OpenBaoMFAMethod, reason string, err error) (ctrl.Result, error) {
	before := object.Status.DeepCopy()
	markError(&object.Status.Conditions, object.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, object, before, &object.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoMFAMethodReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoMFAMethodList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		if list.Items[i].Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return requests
}

func (r *OpenBaoMFAMethodReconciler) mapSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoMFAMethodList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range list.Items {
		method := &list.Items[i]
		refs := []*openbaov1alpha1.SecretKeyReference{method.Spec.SecretKeyRef, method.Spec.APITokenRef, method.Spec.SettingsFileRef}
		for _, ref := range refs {
			if ref != nil && ref.Name == obj.GetName() {
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(method)})
				break
			}
		}
	}
	return requests
}

// SetupWithManager sets up the MFA method controller.
func (r *OpenBaoMFAMethodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoMFAMethod{}).Named("openbao-openbaomfamethod").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecret)).Complete(r)
}

func mfaPolicies(creation openbaov1alpha1.CreationPolicy, deletion openbaov1alpha1.DeletionPolicy) (openbaov1alpha1.CreationPolicy, openbaov1alpha1.DeletionPolicy) {
	if creation == "" {
		creation = openbaov1alpha1.CreationPolicyCreate
	}
	if deletion == "" {
		deletion = openbaov1alpha1.DeletionPolicyOrphan
	}
	return creation, deletion
}

func mfaCreation(object *openbaov1alpha1.OpenBaoMFAMethod) openbaov1alpha1.CreationPolicy {
	creation, _ := mfaPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	return creation
}

func mfaDrift(value *metav1.Duration) time.Duration {
	if value == nil {
		return defaultDriftCheck
	}
	return value.Duration
}

func mfaStrings(values []string, field string) ([]string, error) {
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

func mfaFieldsMatch(desired, observed openbaoclient.AdditionalObject) bool {
	for key, desiredValue := range desired {
		if key == "secret_key" || key == "api_token" || key == "settings_file_base64" {
			continue
		}
		observedValue, ok := observed[key]
		if !ok || !reflectJSONEqual(desiredValue, observedValue) {
			return false
		}
	}
	return true
}

func mfaHash(value openbaoclient.AdditionalObject) string {
	clean := openbaoclient.AdditionalObject{}
	for key, item := range value {
		if key != "secret_key" && key != "api_token" && key != "settings_file_base64" {
			clean[key] = item
		}
	}
	data, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func mfaSecretHash(parts []string) string {
	slices.Sort(parts)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}
