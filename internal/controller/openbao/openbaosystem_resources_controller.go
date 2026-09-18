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
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
)

func mountConnectionRequests(ctx context.Context, c client.Client, obj client.Object, list client.ObjectList, connectionName func(client.Object) string) []reconcile.Request {
	if err := c.List(ctx, list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	switch items := list.(type) {
	case *openbaov1alpha1.OpenBaoAuthMethodList:
		for i := range items.Items {
			item := &items.Items[i]
			if connectionName(item) == obj.GetName() {
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
			}
		}
	case *openbaov1alpha1.OpenBaoSecretEngineList:
		for i := range items.Items {
			item := &items.Items[i]
			if connectionName(item) == obj.GetName() {
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
			}
		}
	}
	return requests
}

func (r *OpenBaoAuthMethodReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoAuthMethodList
	return mountConnectionRequests(ctx, r.Client, obj, &list, func(item client.Object) string {
		return item.(*openbaov1alpha1.OpenBaoAuthMethod).Spec.ConnectionRef.Name
	})
}

func (r *OpenBaoAuthMethodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoAuthMethod{}).
		Named("openbao-openbaoauthmethod").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).
		Complete(r)
}

func (r *OpenBaoSecretEngineReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoSecretEngineList
	return mountConnectionRequests(ctx, r.Client, obj, &list, func(item client.Object) string {
		return item.(*openbaov1alpha1.OpenBaoSecretEngine).Spec.ConnectionRef.Name
	})
}

func (r *OpenBaoSecretEngineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoSecretEngine{}).
		Named("openbao-openbaosecretengine").
		Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).
		Complete(r)
}

// OpenBaoNamespaceReconciler reconciles OpenBao namespaces.
type OpenBaoNamespaceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaonamespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaonamespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaonamespaces/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao namespace.
func (r *OpenBaoNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoNamespace
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{
		Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion,
		DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Path,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Path) == "" {
				return nil, fmt.Errorf("path must not be empty")
			}
			result := map[string]any{}
			if len(object.Spec.CustomMetadata) > 0 {
				result["custom_metadata"] = object.Spec.CustomMetadata
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) {
			value, err := c.GetNamespace(ctx, object.Spec.Path)
			if err != nil {
				return nil, err
			}
			return objectMap(value), nil
		},
		Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteNamespace(ctx, object.Spec.Path, value)
		},
		Delete:       func(c SystemClient) error { return c.DeleteNamespace(ctx, object.Spec.Path) },
		Acquired:     func() bool { return object.Status.Path != "" },
		MarkAcquired: func() { object.Status.Path = object.Spec.Path },
		Observe: func(value map[string]any) {
			object.Status.Path = object.Spec.Path
			if id, ok := value["id"].(string); ok {
				object.Status.ID = id
			}
		},
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoNamespaceReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoNamespaceList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
		}
	}
	return requests
}

func (r *OpenBaoNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoNamespace{}).Named("openbao-openbaonamespace").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

// OpenBaoAuditDeviceReconciler reconciles audit devices.
type OpenBaoAuditDeviceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauditdevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauditdevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoauditdevices/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao audit device.
func (r *OpenBaoAuditDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoAuditDevice
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{
		Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion,
		DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Path,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Type) == "" || strings.TrimSpace(object.Spec.Path) == "" {
				return nil, fmt.Errorf("path and type must not be empty")
			}
			result := map[string]any{systemTypeField: object.Spec.Type}
			if object.Spec.Description != "" {
				result["description"] = object.Spec.Description
			}
			if object.Spec.Local != nil {
				result["local"] = *object.Spec.Local
			}
			if object.Spec.Options != nil {
				result["options"] = object.Spec.Options
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) {
			value, err := c.GetAuditDevice(ctx, object.Spec.Path)
			if err != nil {
				return nil, err
			}
			return objectMap(value), nil
		},
		Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteAuditDevice(ctx, object.Spec.Path, value)
		},
		Delete:       func(c SystemClient) error { return c.DeleteAuditDevice(ctx, object.Spec.Path) },
		Acquired:     func() bool { return object.Status.Path != "" },
		MarkAcquired: func() { object.Status.Path = object.Spec.Path },
		Observe: func(value map[string]any) {
			object.Status.Path = object.Spec.Path
			if typ, ok := value[systemTypeField].(string); ok {
				object.Status.Type = typ
			}
		},
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoAuditDeviceReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoAuditDeviceList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
		}
	}
	return requests
}
func (r *OpenBaoAuditDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoAuditDevice{}).Named("openbao-openbaoauditdevice").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

// OpenBaoRateLimitQuotaReconciler reconciles rate-limit quotas.
type OpenBaoRateLimitQuotaReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoratelimitquotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoratelimitquotas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoratelimitquotas/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao rate-limit quota.
func (r *OpenBaoRateLimitQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoRateLimitQuota
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{
		Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion,
		DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Name,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Type) == "" {
				return nil, fmt.Errorf("type must not be empty")
			}
			result := map[string]any{systemTypeField: object.Spec.Type}
			if object.Spec.Path != "" {
				result["path"] = object.Spec.Path
			}
			if object.Spec.Role != "" {
				result["role"] = object.Spec.Role
			}
			if object.Spec.Rate != "" {
				rate, err := strconv.ParseFloat(object.Spec.Rate, 64)
				if err != nil || rate <= 0 {
					return nil, fmt.Errorf("rate must be a positive number")
				}
				result["rate"] = rate
			}
			if value, err := systemDuration(object.Spec.Interval, "interval"); err != nil {
				return nil, err
			} else if value != nil {
				result["interval"] = value
			}
			if value, err := systemDuration(object.Spec.BlockInterval, "blockInterval"); err != nil {
				return nil, err
			} else if value != nil {
				result["block_interval"] = value
			}
			if object.Spec.Inheritable != nil {
				result["inheritable"] = *object.Spec.Inheritable
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) {
			value, err := c.GetRateLimitQuota(ctx, object.Name)
			return value, err
		},
		Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteRateLimitQuota(ctx, object.Name, value)
		},
		Delete:       func(c SystemClient) error { return c.DeleteRateLimitQuota(ctx, object.Name) },
		Acquired:     func() bool { return object.Status.Name != "" },
		MarkAcquired: func() { object.Status.Name = object.Name },
		Observe:      func(_ map[string]any) { object.Status.Name = object.Name },
		Status:       oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoRateLimitQuotaReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoRateLimitQuotaList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
		}
	}
	return requests
}
func (r *OpenBaoRateLimitQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoRateLimitQuota{}).Named("openbao-openbaoratelimitquota").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

// OpenBaoWorkflowReconciler reconciles workflow definitions.
type OpenBaoWorkflowReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoworkflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoworkflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoworkflows/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao workflow.
func (r *OpenBaoWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoWorkflow
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{
		Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion,
		DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Path,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Path) == "" || object.Spec.Workflow == "" {
				return nil, fmt.Errorf("path and workflow must not be empty")
			}
			result := map[string]any{"workflow": object.Spec.Workflow}
			if object.Spec.Description != "" {
				result["description"] = object.Spec.Description
			}
			if object.Spec.AllowUnauthenticated != nil {
				result["allow_unauthenticated"] = *object.Spec.AllowUnauthenticated
			}
			if object.Spec.CAS != nil {
				result["cas"] = *object.Spec.CAS
			}
			if object.Spec.CASRequired != nil {
				result["cas_required"] = *object.Spec.CASRequired
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) {
			value, err := c.GetWorkflow(ctx, object.Spec.Path)
			return value, err
		},
		Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WriteWorkflow(ctx, object.Spec.Path, value)
		},
		Delete:       func(c SystemClient) error { return c.DeleteWorkflow(ctx, object.Spec.Path) },
		Acquired:     func() bool { return object.Status.Path != "" },
		MarkAcquired: func() { object.Status.Path = object.Spec.Path },
		Observe: func(value map[string]any) {
			object.Status.Path = object.Spec.Path
			if version, ok := value["version"].(float64); ok {
				object.Status.Version = int64(version)
			}
		},
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoWorkflowReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoWorkflowList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
		}
	}
	return requests
}
func (r *OpenBaoWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoWorkflow{}).Named("openbao-openbaoworkflow").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

// OpenBaoPluginReconciler reconciles plugin catalog registrations. The plugin
// binary itself remains external to this operator and must already be present
// in OpenBao's configured plugin directory or OCI registry.
type OpenBaoPluginReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoplugins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoplugins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoplugins/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao plugin catalog registration.
func (r *OpenBaoPluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object openbaov1alpha1.OpenBaoPlugin
	if err := r.Get(ctx, req.NamespacedName, &object); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	before := object.Status.DeepCopy()
	creation, deletion := systemPolicies(object.Spec.CreationPolicy, object.Spec.DeletionPolicy)
	plan := systemPlan{
		Object: &object, ConnectionRef: object.Spec.ConnectionRef, CreationPolicy: creation, DeletionPolicy: deletion,
		DriftInterval: systemDrift(object.Spec.DriftDetectionInterval), Name: object.Spec.Name,
		Desired: func() (map[string]any, error) {
			if strings.TrimSpace(object.Spec.Name) == "" || strings.TrimSpace(object.Spec.Type) == "" {
				return nil, fmt.Errorf("name and type must not be empty")
			}
			result := map[string]any{systemTypeField: object.Spec.Type}
			if object.Spec.Command != "" {
				result["command"] = object.Spec.Command
			}
			if object.Spec.Args != nil {
				result["args"] = object.Spec.Args
			}
			if object.Spec.Env != nil {
				result["env"] = object.Spec.Env
			}
			if object.Spec.SHA256 != "" {
				result["sha256"] = object.Spec.SHA256
			}
			if object.Spec.Version != "" {
				result["version"] = object.Spec.Version
			}
			if object.Spec.OCI != nil {
				result["oci"] = *object.Spec.OCI
			}
			return result, nil
		},
		Get: func(c SystemClient) (map[string]any, error) {
			value, err := c.GetPlugin(ctx, object.Spec.Type, object.Spec.Name)
			return value, err
		},
		Write: func(c SystemClient, value map[string]any, _ bool) error {
			return c.WritePlugin(ctx, object.Spec.Type, object.Spec.Name, value)
		},
		Delete:       func(c SystemClient) error { return c.DeletePlugin(ctx, object.Spec.Type, object.Spec.Name) },
		Acquired:     func() bool { return object.Status.Name != "" },
		MarkAcquired: func() { object.Status.Name = object.Spec.Name },
		Observe: func(value map[string]any) {
			object.Status.Name = object.Spec.Name
			if typ, ok := value[systemTypeField].(string); ok {
				object.Status.Type = typ
			}
		},
		Status: oidcStatusView{ConfigHash: &object.Status.ConfigHash, ObservedGeneration: &object.Status.ObservedGeneration, Conditions: &object.Status.Conditions, Before: before, Current: &object.Status},
	}
	return reconcileSystemNamed(ctx, r.Client, r.ClientCache, r.NewClient, plan)
}

func (r *OpenBaoPluginReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoPluginList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Spec.ConnectionRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
		}
	}
	return requests
}
func (r *OpenBaoPluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoPlugin{}).Named("openbao-openbaoplugin").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}
