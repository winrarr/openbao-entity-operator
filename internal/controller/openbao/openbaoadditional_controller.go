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
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
)

type OpenBaoCORSConfigurationReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaocorsconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaocorsconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaocorsconfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles the OpenBao CORS configuration.
func (r *OpenBaoCORSConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoCORSConfiguration
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Name,
		Desired: func() (map[string]any, error) {
			result := map[string]any{}
			if object.Spec.Enable != nil {
				result["enable"] = *object.Spec.Enable
			}
			if object.Spec.AllowCredentials != nil {
				result["allow_credentials"] = *object.Spec.AllowCredentials
			}
			if object.Spec.AllowedHeaders != nil {
				result["allowed_headers"] = append([]string(nil), object.Spec.AllowedHeaders...)
			}
			if object.Spec.AllowedOrigins != nil {
				result["allowed_origins"] = append([]string(nil), object.Spec.AllowedOrigins...)
			}
			return result, nil
		},
		Get:      func(c SystemClient) (map[string]any, error) { return c.GetCORSConfiguration(ctx) },
		Write:    func(c SystemClient, value map[string]any, _ bool) error { return c.WriteCORSConfiguration(ctx, value) },
		Delete:   func(c SystemClient) error { return c.DeleteCORSConfiguration(ctx) },
		Acquired: func() bool { return object.Status.ConfigHash != "" }, MarkAcquired: func() {}, Observe: func(_ map[string]any) {},
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoCORSConfigurationReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoCORSConfigurationList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoCORSConfiguration) string { return item.Spec.ConnectionRef.Name })
}

// SetupWithManager sets up the CORS configuration controller.
func (r *OpenBaoCORSConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoCORSConfiguration{}).Named("openbao-openbaocorsconfiguration").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

type OpenBaoAuditRequestHeaderReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauditrequestheaders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauditrequestheaders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauditrequestheaders/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao audit request header.
func (r *OpenBaoAuditRequestHeaderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoAuditRequestHeader
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Header,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Header) == "" {
				return nil, fmt.Errorf("header must not be empty")
			}
			result := map[string]any{}
			if object.Spec.HMAC != nil {
				result["hmac"] = *object.Spec.HMAC
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) { return c.GetAuditRequestHeader(ctx, object.Spec.Header) }, Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteAuditRequestHeader(ctx, object.Spec.Header, value)
		}, Delete: func(c SystemClient) error { return c.DeleteAuditRequestHeader(ctx, object.Spec.Header) },
		Acquired: func() bool { return object.Status.Header != "" }, MarkAcquired: func() { object.Status.Header = object.Spec.Header }, Observe: func(_ map[string]any) { object.Status.Header = object.Spec.Header },
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status}}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoAuditRequestHeaderReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoAuditRequestHeaderList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoAuditRequestHeader) string { return item.Spec.ConnectionRef.Name })
}

// SetupWithManager sets up the audit request-header controller.
func (r *OpenBaoAuditRequestHeaderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoAuditRequestHeader{}).Named("openbao-openbaoauditrequestheader").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

type OpenBaoUIHeaderReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaouiheaders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaouiheaders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaouiheaders/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao UI header.
func (r *OpenBaoUIHeaderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoUIHeader
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Header,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Header) == "" {
				return nil, fmt.Errorf("header must not be empty")
			}
			result := map[string]any{}
			if object.Spec.Values != nil {
				result["values"] = append([]string(nil), object.Spec.Values...)
			}
			if object.Spec.Multivalue != nil {
				result["multivalue"] = *object.Spec.Multivalue
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) { return c.GetUIHeader(ctx, object.Spec.Header) }, Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteUIHeader(ctx, object.Spec.Header, value)
		}, Delete: func(c SystemClient) error { return c.DeleteUIHeader(ctx, object.Spec.Header) },
		Acquired: func() bool { return object.Status.Header != "" }, MarkAcquired: func() { object.Status.Header = object.Spec.Header }, Observe: func(_ map[string]any) { object.Status.Header = object.Spec.Header },
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status}}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoUIHeaderReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoUIHeaderList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoUIHeader) string { return item.Spec.ConnectionRef.Name })
}

// SetupWithManager sets up the UI header controller.
func (r *OpenBaoUIHeaderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoUIHeader{}).Named("openbao-openbaouiheader").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

type OpenBaoRateLimitQuotaConfigurationReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoratelimitquotaconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoratelimitquotaconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoratelimitquotaconfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles global OpenBao rate-limit quota configuration.
func (r *OpenBaoRateLimitQuotaConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoRateLimitQuotaConfiguration
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Name,
		Desired: func() (map[string]any, error) {
			result := map[string]any{}
			if object.Spec.EnableRateLimitAuditLogging != nil {
				result["enable_rate_limit_audit_logging"] = *object.Spec.EnableRateLimitAuditLogging
			}
			if object.Spec.EnableRateLimitResponseHeaders != nil {
				result["enable_rate_limit_response_headers"] = *object.Spec.EnableRateLimitResponseHeaders
			}
			if object.Spec.RateLimitExemptPaths != nil {
				result["rate_limit_exempt_paths"] = append([]string(nil), object.Spec.RateLimitExemptPaths...)
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) { return c.GetRateLimitQuotaConfiguration(ctx) }, Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteRateLimitQuotaConfiguration(ctx, value)
		}, Delete: func(SystemClient) error { return unsupportedConfigurationDelete("rate-limit quota configuration") },
		Acquired: func() bool { return object.Status.ConfigHash != "" }, MarkAcquired: func() {}, Observe: func(_ map[string]any) {}, Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status}}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoRateLimitQuotaConfigurationReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoRateLimitQuotaConfigurationList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoRateLimitQuotaConfiguration) string {
		return item.Spec.ConnectionRef.Name
	})
}

// SetupWithManager sets up the global quota configuration controller.
func (r *OpenBaoRateLimitQuotaConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoRateLimitQuotaConfiguration{}).Named("openbao-openbaoratelimitquotaconfiguration").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

type OpenBaoLoggerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaologgers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaologgers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaologgers/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles OpenBao logger verbosity.
func (r *OpenBaoLoggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoLogger
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Name,
		Desired: func() (map[string]any, error) {
			switch object.Spec.Level {
			case "trace", "debug", "info", "warn", "error":
				return map[string]any{"level": object.Spec.Level}, nil
			default:
				return nil, fmt.Errorf("level must be one of trace, debug, info, warn, or error")
			}
		},
		Get: func(c SystemClient) (map[string]any, error) { return c.GetLogger(ctx, object.Spec.Name) }, Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteLogger(ctx, object.Spec.Name, value)
		}, Delete: func(c SystemClient) error { return c.DeleteLogger(ctx, object.Spec.Name) },
		Acquired: func() bool { return object.Status.ConfigHash != "" }, MarkAcquired: func() {}, Observe: func(value map[string]any) {
			object.Status.Name = object.Spec.Name
			if level, ok := value["level"].(string); ok {
				object.Status.Level = level
			}
		}, Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status}}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoLoggerReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoLoggerList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoLogger) string { return item.Spec.ConnectionRef.Name })
}

// SetupWithManager sets up the logger controller.
func (r *OpenBaoLoggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoLogger{}).Named("openbao-openbaologger").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

type OpenBaoEncryptionKeyConfigurationReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}
type OpenBaoKeyringRotationConfigurationReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoencryptionkeyconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoencryptionkeyconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoencryptionkeyconfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaokeyringrotationconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaokeyringrotationconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaokeyringrotationconfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles automatic encryption-key rotation configuration.
func (r *OpenBaoEncryptionKeyConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoEncryptionKeyConfiguration
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return reconcileRotationConfiguration(ctx, r.Client, r.ClientCache, r.NewClient, &object, object.Spec, object.Status.DeepCopy(), func(c SystemClient, value map[string]any) error { return c.WriteEncryptionKeyConfiguration(ctx, value) }, func(c SystemClient) (map[string]any, error) { return c.GetEncryptionKeyConfiguration(ctx) })
}

func (r *OpenBaoEncryptionKeyConfigurationReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoEncryptionKeyConfigurationList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoEncryptionKeyConfiguration) string {
		return item.Spec.ConnectionRef.Name
	})
}

// SetupWithManager sets up the encryption-key rotation configuration controller.
func (r *OpenBaoEncryptionKeyConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoEncryptionKeyConfiguration{}).Named("openbao-openbaoencryptionkeyconfiguration").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

// Reconcile reconciles automatic keyring rotation configuration.
func (r *OpenBaoKeyringRotationConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoKeyringRotationConfiguration
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return reconcileRotationConfiguration(ctx, r.Client, r.ClientCache, r.NewClient, &object, object.Spec, object.Status.DeepCopy(), func(c SystemClient, value map[string]any) error {
		return c.WriteKeyringRotationConfiguration(ctx, value)
	}, func(c SystemClient) (map[string]any, error) { return c.GetKeyringRotationConfiguration(ctx) })
}

func (r *OpenBaoKeyringRotationConfigurationReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoKeyringRotationConfigurationList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	return connectionRequests(obj, objectPointers(list.Items), func(item *openbaov1alpha1.OpenBaoKeyringRotationConfiguration) string {
		return item.Spec.ConnectionRef.Name
	})
}

// SetupWithManager sets up the keyring rotation configuration controller.
func (r *OpenBaoKeyringRotationConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoKeyringRotationConfiguration{}).Named("openbao-openbaokeyringrotationconfiguration").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

func rotationDesired(spec openbaov1alpha1.OpenBaoRotationConfigurationSpec) (map[string]any, error) {
	result := map[string]any{}
	if spec.Enabled != nil {
		result["enabled"] = *spec.Enabled
	}
	if spec.Interval != nil {
		seconds, err := durationSeconds(spec.Interval, "interval")
		if err != nil {
			return nil, err
		}
		result["interval"] = seconds
	}
	if spec.MaxOperations != nil {
		if *spec.MaxOperations < 0 {
			return nil, fmt.Errorf("maxOperations must not be negative")
		}
		result["max_operations"] = *spec.MaxOperations
	}
	return result, nil
}

func reconcileRotationConfiguration(ctx context.Context, kubeClient client.Client, cache *ConnectionClientCache, newClient func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error), object client.Object, spec openbaov1alpha1.OpenBaoRotationConfigurationSpec, before any, write func(SystemClient, map[string]any) error, get func(SystemClient) (map[string]any, error)) (ctrl.Result, error) {
	beforeStatus := before
	desired, err := rotationDesired(spec)
	if err != nil {
		return ctrl.Result{}, err
	}
	creation, deletion := systemPolicies(spec.CreationPolicy, spec.DeletionPolicy)
	plan := systemPlan{Object: object, ConnectionRef: spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion, DriftInterval: systemDrift(spec.DriftDetectionInterval), Name: object.GetName(), Desired: func() (map[string]any, error) { return desired, nil }, Get: func(c SystemClient) (map[string]any, error) { return get(c) }, Write: func(c SystemClient, value map[string]any, _ bool) error { return write(c, value) }, Delete: func(SystemClient) error { return unsupportedConfigurationDelete("rotation configuration") }, Acquired: func() bool { return false }, MarkAcquired: func() {}, Observe: func(_ map[string]any) {}, Status: rotationStatusView(object, beforeStatus)}
	return reconcileSystemNamed(ctx, kubeClient, cache, newClient, plan)
}

func rotationStatusView(object client.Object, before any) oidcStatusView {
	switch item := object.(type) {
	case *openbaov1alpha1.OpenBaoEncryptionKeyConfiguration:
		return oidcStatusView{ConfigHash: &item.Status.ConfigHash, ObservedGeneration: &item.Status.ObservedGeneration, Conditions: &item.Status.Conditions, Before: before, Current: &item.Status}
	case *openbaov1alpha1.OpenBaoKeyringRotationConfiguration:
		return oidcStatusView{ConfigHash: &item.Status.ConfigHash, ObservedGeneration: &item.Status.ObservedGeneration, Conditions: &item.Status.Conditions, Before: before, Current: &item.Status}
	default:
		panic("unsupported rotation configuration")
	}
}

func unsupportedConfigurationDelete(name string) error {
	return fmt.Errorf("OpenBao does not expose deletion for %s; use deletionPolicy=Orphan", name)
}

func objectPointers[T any](items []T) []*T {
	result := make([]*T, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}

func connectionRequests[T client.Object](obj client.Object, items []T, connection func(T) string) []reconcile.Request {
	requests := make([]reconcile.Request, 0)
	for _, item := range items {
		if connection(item) == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
		}
	}
	return requests
}
