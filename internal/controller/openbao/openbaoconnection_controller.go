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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

// ConnectionClient is the OpenBao surface used by OpenBaoConnection.
type ConnectionClient interface {
	CheckHealth(context.Context) (*openbaoclient.Health, error)
	LookupSelf(context.Context) error
}

// OpenBaoConnectionReconciler reconciles an OpenBaoConnection object.
type OpenBaoConnectionReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (ConnectionClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *OpenBaoConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("openbao-connection")
	var connection openbaov1alpha1.OpenBaoConnection
	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := connection.Status.DeepCopy()
	connection.Status.ObservedGeneration = connection.Generation

	apiClient, err := r.clientFor(ctx, &connection)
	if err != nil {
		connection.Status.Authenticated = false
		markStalled(&connection.Status.Conditions, connection.Generation, "InvalidConfiguration", err)
		if statusErr := updateStatusIfChanged(ctx, r.Client, &connection, before, &connection.Status); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: dependencyRetry}, nil
	}

	if connection.Spec.Namespace == "" {
		health, err := apiClient.CheckHealth(ctx)
		if err != nil {
			return r.fail(ctx, &connection, "HealthCheckFailed", err)
		}
		connection.Status.Version = health.Version
		connection.Status.Initialized = health.Initialized
		connection.Status.Sealed = health.Sealed
		connection.Status.Standby = health.Standby
	} else {
		// OpenBao does not expose sys/health from within a namespace. The
		// namespaced token self-lookup below verifies both reachability and
		// authentication without losing the namespace boundary.
		logger.Info("Skipping namespace-scoped OpenBao health check", "namespace", connection.Spec.Namespace)
	}
	if err := apiClient.LookupSelf(ctx); err != nil {
		return r.fail(ctx, &connection, "AuthenticationFailed", err)
	}

	connection.Status.Authenticated = true
	readyMessage := "OpenBao is reachable and configured authentication succeeded"
	if connection.Spec.Namespace != "" {
		readyMessage = "Configured authentication succeeded in the OpenBao namespace"
	}
	markReady(&connection.Status.Conditions, connection.Generation, readyMessage)
	if err := updateStatusIfChanged(ctx, r.Client, &connection, before, &connection.Status); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Verified OpenBao connection", "address", connection.Spec.Address, "namespace", connection.Spec.Namespace, "version", connection.Status.Version)
	return ctrl.Result{RequeueAfter: defaultDriftCheck}, nil

}

func (r *OpenBaoConnectionReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (ConnectionClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoConnectionReconciler) fail(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection, reason string, err error) (ctrl.Result, error) {
	before := connection.Status.DeepCopy()
	connection.Status.Authenticated = false
	markError(&connection.Status.Conditions, connection.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, connection, before, &connection.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoConnectionReconciler) mapSecretToConnections(ctx context.Context, obj client.Object) []reconcile.Request {
	var connections openbaov1alpha1.OpenBaoConnectionList
	if err := r.List(ctx, &connections, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range connections.Items {
		connection := &connections.Items[i]
		if (connection.Spec.TokenSecretRef != nil && connection.Spec.TokenSecretRef.Name == obj.GetName()) ||
			(connection.Spec.CABundleSecretRef != nil && connection.Spec.CABundleSecretRef.Name == obj.GetName()) {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(connection)})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenBaoConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openbaov1alpha1.OpenBaoConnection{}).
		Named("openbao-openbaoconnection").
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToConnections)).
		Complete(r)
}
