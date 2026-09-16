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
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

// ConnectionClientCache keeps Kubernetes Auth clients, and therefore their
// in-memory OpenBao token leases, alive across reconciliations. Static-token
// clients are intentionally not cached so a rotated Secret is used immediately.
type ConnectionClientCache struct {
	mu      sync.Mutex
	clients map[types.NamespacedName]cachedConnectionClient
}

type cachedConnectionClient struct {
	key    kubernetesClientKey
	client *openbaoclient.Client
}

type kubernetesClientKey struct {
	connection types.NamespacedName
	address    string
	namespace  string
	mountPath  string
	role       string
	timeout    time.Duration
	caDigest   [sha256.Size]byte
}

// NewConnectionClientCache returns a cache for connection-scoped clients.
func NewConnectionClientCache() *ConnectionClientCache {
	return &ConnectionClientCache{
		clients: make(map[types.NamespacedName]cachedConnectionClient),
	}
}

const (
	finalizerName      = "openbao.openbao-operator.io/finalizer"
	dependencyRetry    = 15 * time.Second
	defaultDriftCheck  = 2 * time.Minute
	conditionReady     = "Ready"
	conditionStalled   = "Stalled"
	reasonReconciled   = "Reconciled"
	reasonReconcileErr = "ReconcileError"
)

func setCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	for i := range *conditions {
		current := &(*conditions)[i]
		if current.Type != condition.Type {
			continue
		}
		if current.Status == condition.Status && current.Reason == condition.Reason &&
			current.Message == condition.Message && current.ObservedGeneration == condition.ObservedGeneration {
			return
		}
		*current = condition
		return
	}
	*conditions = append(*conditions, condition)
}

func markReady(conditions *[]metav1.Condition, generation int64, message string) {
	setCondition(conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             reasonReconciled,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	})
	setCondition(conditions, metav1.Condition{
		Type:               conditionStalled,
		Status:             metav1.ConditionFalse,
		Reason:             "NotStalled",
		Message:            "The resource is not stalled",
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	})
}

func markError(conditions *[]metav1.Condition, generation int64, reason string, err error) {
	message := "OpenBao reconciliation failed"
	if err != nil {
		message = err.Error()
	}
	setCondition(conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	})
}

func markStalled(conditions *[]metav1.Condition, generation int64, reason string, err error) {
	markError(conditions, generation, reason, err)
	message := "The resource is blocked until its dependencies or configuration are fixed"
	if err != nil {
		message = err.Error()
	}
	setCondition(conditions, metav1.Condition{
		Type:               conditionStalled,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	})
}

func updateStatusIfChanged(ctx context.Context, kubeClient client.Client, obj client.Object, before, after any) error {
	if reflect.DeepEqual(before, after) {
		return nil
	}
	return kubeClient.Status().Update(ctx, obj)
}

func connectionClientFor(ctx context.Context, kubeClient client.Client, connection *openbaov1alpha1.OpenBaoConnection) (*openbaoclient.Client, error) {
	timeout := connectionRequestTimeout(connection)
	caBundle, err := connectionCABundle(ctx, kubeClient, connection)
	if err != nil {
		return nil, err
	}

	if auth := connection.Spec.KubernetesAuth; auth != nil {
		return openbaoclient.NewWithKubernetesAuth(connection.Spec.Address, openbaoclient.KubernetesAuthOptions{
			MountPath: auth.MountPath,
			Role:      auth.Role,
			JWTSource: openbaoclient.ServiceAccountTokenSource,
		}, timeout, caBundle, connection.Spec.Namespace)
	}
	if connection.Spec.TokenSecretRef == nil {
		return nil, fmt.Errorf("OpenBaoConnection must configure tokenSecretRef or kubernetesAuth")
	}

	tokenSecretRef := connection.Spec.TokenSecretRef
	var secret corev1.Secret
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: connection.Namespace, Name: tokenSecretRef.Name}, &secret); err != nil {
		return nil, fmt.Errorf("read token Secret %s/%s: %w", connection.Namespace, tokenSecretRef.Name, err)
	}
	tokenKey := tokenSecretRef.Key
	if tokenKey == "" {
		tokenKey = "token"
	}
	token := strings.TrimSpace(string(secret.Data[tokenKey]))
	if token == "" {
		return nil, fmt.Errorf("token Secret %s/%s has no non-empty %q key", connection.Namespace, connection.Spec.TokenSecretRef.Name, tokenKey)
	}

	return openbaoclient.NewWithNamespace(connection.Spec.Address, token, timeout, caBundle, connection.Spec.Namespace)
}

func connectionRequestTimeout(connection *openbaov1alpha1.OpenBaoConnection) time.Duration {
	if connection.Spec.RequestTimeout != nil && connection.Spec.RequestTimeout.Duration > 0 {
		return connection.Spec.RequestTimeout.Duration
	}
	return 30 * time.Second
}

func connectionCABundle(ctx context.Context, kubeClient client.Client, connection *openbaov1alpha1.OpenBaoConnection) ([]byte, error) {
	ref := connection.Spec.CABundleSecretRef
	if ref == nil {
		return nil, nil
	}

	var caSecret corev1.Secret
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: connection.Namespace, Name: ref.Name}, &caSecret); err != nil {
		return nil, fmt.Errorf("read CA bundle Secret %s/%s: %w", connection.Namespace, ref.Name, err)
	}
	caKey := ref.Key
	if caKey == "" {
		caKey = "ca.crt"
	}
	caBundle := caSecret.Data[caKey]
	if len(caBundle) == 0 {
		return nil, fmt.Errorf("CA bundle Secret %s/%s has no %q key", connection.Namespace, ref.Name, caKey)
	}
	return caBundle, nil
}

// ClientFor returns the connection client, retaining Kubernetes Auth token
// leases across reconciliations while replacing clients when their config or
// CA bundle changes.
func (c *ConnectionClientCache) ClientFor(ctx context.Context, kubeClient client.Client, connection *openbaov1alpha1.OpenBaoConnection) (*openbaoclient.Client, error) {
	if connection.Spec.KubernetesAuth == nil {
		return connectionClientFor(ctx, kubeClient, connection)
	}

	caBundle, err := connectionCABundle(ctx, kubeClient, connection)
	if err != nil {
		return nil, err
	}
	auth := connection.Spec.KubernetesAuth
	key := kubernetesClientKey{
		connection: types.NamespacedName{Namespace: connection.Namespace, Name: connection.Name},
		address:    connection.Spec.Address,
		namespace:  connection.Spec.Namespace,
		mountPath:  auth.MountPath,
		role:       auth.Role,
		timeout:    connectionRequestTimeout(connection),
		caDigest:   sha256.Sum256(caBundle),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clients == nil {
		c.clients = make(map[types.NamespacedName]cachedConnectionClient)
	}
	if cached, ok := c.clients[key.connection]; ok && cached.key == key {
		return cached.client, nil
	}

	apiClient, err := openbaoclient.NewWithKubernetesAuth(connection.Spec.Address, openbaoclient.KubernetesAuthOptions{
		MountPath: auth.MountPath,
		Role:      auth.Role,
		JWTSource: openbaoclient.ServiceAccountTokenSource,
	}, key.timeout, caBundle, connection.Spec.Namespace)
	if err != nil {
		return nil, err
	}
	c.clients[key.connection] = cachedConnectionClient{key: key, client: apiClient}
	return apiClient, nil
}

func resolveConnection(ctx context.Context, kubeClient client.Client, namespace string, ref openbaov1alpha1.OpenBaoConnectionReference) (*openbaov1alpha1.OpenBaoConnection, error) {
	var connection openbaov1alpha1.OpenBaoConnection
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &connection); err != nil {
		return nil, err
	}
	return &connection, nil
}

func resolveEntity(ctx context.Context, kubeClient client.Client, namespace string, ref openbaov1alpha1.OpenBaoEntityReference) (*openbaov1alpha1.OpenBaoEntity, error) {
	var entity openbaov1alpha1.OpenBaoEntity
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func resolveGroup(ctx context.Context, kubeClient client.Client, namespace string, ref openbaov1alpha1.OpenBaoGroupReference) (*openbaov1alpha1.OpenBaoGroup, error) {
	var group openbaov1alpha1.OpenBaoGroup
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func ensureFinalizer(ctx context.Context, kubeClient client.Client, obj client.Object) error {
	if controllerutil.ContainsFinalizer(obj, finalizerName) {
		return nil
	}
	controllerutil.AddFinalizer(obj, finalizerName)
	return kubeClient.Update(ctx, obj)
}

func removeFinalizer(ctx context.Context, kubeClient client.Client, obj client.Object) error {
	if !controllerutil.ContainsFinalizer(obj, finalizerName) {
		return nil
	}
	controllerutil.RemoveFinalizer(obj, finalizerName)
	return kubeClient.Update(ctx, obj)
}

func removeFinalizerAfterDependencyLoss(ctx context.Context, kubeClient client.Client, obj client.Object, dependency string, err error) (ctrl.Result, error) {
	log.FromContext(ctx).Error(err, "Releasing deletion finalizer because cleanup dependency is unavailable", "dependency", dependency, "resource", client.ObjectKeyFromObject(obj))
	return ctrl.Result{}, removeFinalizer(ctx, kubeClient, obj)
}

func normalizedPolicies(policies []string) []string {
	result := append([]string(nil), policies...)
	slices.Sort(result)
	return result
}

func desiredEntityRequest(entity *openbaov1alpha1.OpenBaoEntity) openbaoclient.EntityRequest {
	metadata := maps.Clone(entity.Spec.Metadata)
	return openbaoclient.EntityRequest{
		Name:     entity.Name,
		Metadata: metadata,
		Policies: normalizedPolicies(entity.Spec.Policies),
		Disabled: entity.Spec.Disabled,
	}
}

func entityMatches(desired openbaoclient.EntityRequest, current *openbaoclient.Entity) bool {
	return current.Name == desired.Name && current.Disabled == desired.Disabled &&
		reflect.DeepEqual(current.Metadata, desired.Metadata) &&
		reflect.DeepEqual(normalizedPolicies(current.Policies), desired.Policies)
}

func observeEntityStatus(entity *openbaov1alpha1.OpenBaoEntity, observed *openbaoclient.Entity) {
	entity.Status.ID = observed.ID
	entity.Status.Name = observed.Name
	entity.Status.Metadata = copyStringMap(observed.Metadata)
	entity.Status.Policies = normalizedPolicies(observed.Policies)
	entity.Status.Disabled = observed.Disabled
	entity.Status.ObservedGeneration = entity.Generation
}

func copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	return maps.Clone(source)
}

func entityDriftInterval(entity *openbaov1alpha1.OpenBaoEntity) time.Duration {
	if entity.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return entity.Spec.DriftDetectionInterval.Duration
}

func creationPolicy(entity *openbaov1alpha1.OpenBaoEntity) openbaov1alpha1.CreationPolicy {
	if entity.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return entity.Spec.CreationPolicy
}

func deletionPolicy(entity *openbaov1alpha1.OpenBaoEntity) openbaov1alpha1.DeletionPolicy {
	if entity.Spec.DeletionPolicy == "" {
		return openbaov1alpha1.DeletionPolicyOrphan
	}
	return entity.Spec.DeletionPolicy
}

func isNotFound(err error) bool {
	return apierrors.IsNotFound(err) || openbaoclient.IsNotFound(err)
}

func dependencyMessage(resource string, name types.NamespacedName, err error) error {
	if err == nil {
		return fmt.Errorf("%s %s is not ready", resource, name)
	}
	return fmt.Errorf("%s %s is not ready: %w", resource, name, err)
}
