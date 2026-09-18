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
	"maps"
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

type PersonaClient interface {
	ListPersonaIDs(context.Context) ([]string, error)
	GetPersonaByID(context.Context, string) (*openbaoclient.Persona, error)
	CreatePersona(context.Context, openbaoclient.PersonaRequest) (string, error)
	UpdatePersona(context.Context, string, openbaoclient.PersonaRequest) (*openbaoclient.Persona, error)
	DeletePersona(context.Context, string) error
}

type OpenBaoPersonaReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	NewClient   func(context.Context, *openbaov1alpha1.OpenBaoConnection) (PersonaClient, error)
	ClientCache *ConnectionClientCache
}

// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopersonas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopersonas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaopersonas/finalizers,verbs=update
// +kubebuilder:rbac:groups=openbao.openbao-operator.io,resources=openbaoconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles an OpenBao persona.
func (r *OpenBaoPersonaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var persona openbaov1alpha1.OpenBaoPersona
	if err := r.Get(ctx, req.NamespacedName, &persona); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if persona.DeletionTimestamp.IsZero() && persona.Spec.DeletionPolicy.IsDelete() {
		if err := ensureFinalizer(ctx, r.Client, &persona); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !persona.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &persona)
	}

	before := persona.Status.DeepCopy()
	persona.Status.ObservedGeneration = persona.Generation
	connection, err := resolveConnection(ctx, r.Client, persona.Namespace, persona.Spec.ConnectionRef)
	if err != nil {
		return r.dependencyFailure(ctx, &persona, fmt.Errorf("read OpenBaoConnection %s/%s: %w", persona.Namespace, persona.Spec.ConnectionRef.Name, err))
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, conditionReady) {
		return r.dependencyFailure(ctx, &persona, dependencyMessage("OpenBaoConnection", client.ObjectKeyFromObject(connection), nil))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return r.dependencyFailure(ctx, &persona, err)
	}
	observed, err := r.acquire(ctx, &persona, apiClient)
	if err != nil {
		return r.fail(ctx, &persona, "PersonaAcquireFailed", err)
	}
	desired := personaRequest(&persona)
	if !personaMatches(desired, observed) {
		observed, err = apiClient.UpdatePersona(ctx, persona.Status.ID, desired)
		if err != nil {
			return r.fail(ctx, &persona, "PersonaUpdateFailed", err)
		}
		if observed == nil {
			return r.fail(ctx, &persona, "InvalidOpenBaoResponse", fmt.Errorf("OpenBao returned no persona after updating %q", persona.Status.ID))
		}
	}
	if observed.ID == "" {
		observed.ID = persona.Status.ID
	}
	if observed.ID != persona.Status.ID {
		return r.fail(ctx, &persona, "PersonaIdentityMismatch", fmt.Errorf("OpenBao persona identity changed from %q to %q", persona.Status.ID, observed.ID))
	}
	persona.Status.ID = observed.ID
	persona.Status.Name = observed.Name
	persona.Status.EntityID = observed.EntityID
	persona.Status.MountAccessor = observed.MountAccessor
	persona.Status.Metadata = observed.Metadata
	persona.Status.ConfigHash = personaHash(observed)
	markReady(&persona.Status.Conditions, persona.Generation, "OpenBao persona is reconciled")
	if err := updateStatusIfChanged(ctx, r.Client, &persona, before, &persona.Status); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: personaDrift(&persona)}, nil
}

func (r *OpenBaoPersonaReconciler) clientFor(ctx context.Context, connection *openbaov1alpha1.OpenBaoConnection) (PersonaClient, error) {
	if r.NewClient != nil {
		return r.NewClient(ctx, connection)
	}
	if r.ClientCache != nil {
		return r.ClientCache.ClientFor(ctx, r.Client, connection)
	}
	return connectionClientFor(ctx, r.Client, connection)
}

func (r *OpenBaoPersonaReconciler) acquire(ctx context.Context, persona *openbaov1alpha1.OpenBaoPersona, apiClient PersonaClient) (*openbaoclient.Persona, error) {
	if persona.Status.ID != "" {
		observed, err := apiClient.GetPersonaByID(ctx, persona.Status.ID)
		if err == nil {
			return observed, nil
		}
		if !isNotFound(err) {
			return nil, err
		}
		persona.Status.ID = ""
	}
	ids, err := apiClient.ListPersonaIDs(ctx)
	if err != nil && !isNotFound(err) {
		return nil, err
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		candidate, readErr := apiClient.GetPersonaByID(ctx, id)
		if readErr != nil {
			if isNotFound(readErr) {
				continue
			}
			return nil, readErr
		}
		if candidate.Name == persona.Spec.Name && candidate.EntityID == persona.Spec.EntityID && candidate.MountAccessor == persona.Spec.MountAccessor {
			if !personaCreation(persona).AllowsAdoption() {
				return nil, fmt.Errorf("OpenBao persona %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", persona.Spec.Name)
			}
			persona.Status.ID = id
			return candidate, nil
		}
	}
	if !personaCreation(persona).AllowsCreation() {
		return nil, fmt.Errorf("OpenBao persona %q does not exist and creationPolicy=%s does not allow creation", persona.Spec.Name, personaCreation(persona))
	}
	id, err := apiClient.CreatePersona(ctx, personaRequest(persona))
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("OpenBao returned an empty ID when creating persona %q", persona.Spec.Name)
	}
	persona.Status.ID = id
	return apiClient.GetPersonaByID(ctx, id)
}

func (r *OpenBaoPersonaReconciler) reconcileDeletion(ctx context.Context, persona *openbaov1alpha1.OpenBaoPersona) (ctrl.Result, error) {
	if persona.Spec.DeletionPolicy != openbaov1alpha1.DeletionPolicyDelete || persona.Status.ID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, persona)
	}
	before := persona.Status.DeepCopy()
	connection, err := resolveConnection(ctx, r.Client, persona.Namespace, persona.Spec.ConnectionRef)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, persona, before, &persona.Status, &persona.Status.Conditions, persona.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, persona, "OpenBaoConnection", err))
	}
	apiClient, err := r.clientFor(ctx, connection)
	if err != nil {
		return recordCleanupFailure(ctx, r.Client, persona, before, &persona.Status, &persona.Status.Conditions, persona.Generation, reasonCleanupDependencyUnavailable, cleanupDependencyError(ctx, persona, "OpenBaoConnection credentials", err))
	}
	if err := apiClient.DeletePersona(ctx, persona.Status.ID); err != nil && !isNotFound(err) {
		return recordCleanupFailure(ctx, r.Client, persona, before, &persona.Status, &persona.Status.Conditions, persona.Generation, "CleanupFailed", err)
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, persona)
}

func (r *OpenBaoPersonaReconciler) dependencyFailure(ctx context.Context, persona *openbaov1alpha1.OpenBaoPersona, err error) (ctrl.Result, error) {
	before := persona.Status.DeepCopy()
	markStalled(&persona.Status.Conditions, persona.Generation, "DependencyNotReady", err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, persona, before, &persona.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: dependencyRetry}, nil
}

func (r *OpenBaoPersonaReconciler) fail(ctx context.Context, persona *openbaov1alpha1.OpenBaoPersona, reason string, err error) (ctrl.Result, error) {
	before := persona.Status.DeepCopy()
	markError(&persona.Status.Conditions, persona.Generation, reason, err)
	if statusErr := updateStatusIfChanged(ctx, r.Client, persona, before, &persona.Status); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *OpenBaoPersonaReconciler) mapConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	var list openbaov1alpha1.OpenBaoPersonaList
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

// SetupWithManager sets up the persona controller.
func (r *OpenBaoPersonaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&openbaov1alpha1.OpenBaoPersona{}).Named("openbao-openbaopersona").Watches(&openbaov1alpha1.OpenBaoConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnection)).Complete(r)
}

func personaRequest(persona *openbaov1alpha1.OpenBaoPersona) openbaoclient.PersonaRequest {
	return openbaoclient.PersonaRequest{Name: persona.Spec.Name, EntityID: persona.Spec.EntityID, MountAccessor: persona.Spec.MountAccessor, Metadata: persona.Spec.Metadata}
}

func personaMatches(desired openbaoclient.PersonaRequest, observed *openbaoclient.Persona) bool {
	return observed != nil && desired.Name == observed.Name && desired.EntityID == observed.EntityID && desired.MountAccessor == observed.MountAccessor && maps.Equal(desired.Metadata, observed.Metadata)
}

func personaHash(persona *openbaoclient.Persona) string {
	data, err := json.Marshal(persona)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func personaCreation(persona *openbaov1alpha1.OpenBaoPersona) openbaov1alpha1.CreationPolicy {
	if persona.Spec.CreationPolicy == "" {
		return openbaov1alpha1.CreationPolicyCreate
	}
	return persona.Spec.CreationPolicy
}

func personaDrift(persona *openbaov1alpha1.OpenBaoPersona) time.Duration {
	if persona.Spec.DriftDetectionInterval == nil {
		return defaultDriftCheck
	}
	return persona.Spec.DriftDetectionInterval.Duration
}
