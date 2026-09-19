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
	"maps"
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openbaov1alpha1 "github.com/rkthtrifork/openbao-entity-operator/api/openbao/v1alpha1"
	"github.com/rkthtrifork/openbao-entity-operator/internal/openbaoclient"
)

const (
	testOIDCProviderName = "provider"
	testOIDCClientName   = "client"
	testOIDCIssuer       = "https://issuer.example.test"
	testOIDCIssuerField  = "issuer"
	testLoggerLevelField = "level"
	testPersonaMount     = "auth_kubernetes"
)

func TestSystemReconcilerConformance(t *testing.T) {
	t.Run("creates and corrects drift", func(t *testing.T) {
		connection := readyConnection()
		logger := &openbaov1alpha1.OpenBaoLogger{
			ObjectMeta: metav1.ObjectMeta{Name: testLoggerResourceName, Namespace: testNamespace},
			Spec: openbaov1alpha1.OpenBaoLoggerSpec{
				ConnectionRef: connectionReference(connection),
				Name:          testLoggerName,
				Level:         testLoggerLevel,
			},
		}
		kubeClient := newTestClient(connection, logger)
		baoClient := &fakeSystemClient{}
		reconciler := newLoggerReconciler(kubeClient, baoClient)
		request := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(logger)}

		if result, err := reconciler.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		} else if result.RequeueAfter <= 0 {
			t.Fatalf("RequeueAfter = %s, want periodic drift check", result.RequeueAfter)
		}
		if baoClient.loggerWriteCalls != 1 || baoClient.loggers[testLoggerName][testLoggerLevelField] != testLoggerLevel {
			t.Fatalf("writes = %d, loggers = %#v, want one debug write", baoClient.loggerWriteCalls, baoClient.loggers)
		}

		baoClient.loggers[testLoggerName] = openbaoclient.AdditionalObject{testLoggerLevelField: testLoggerInfoLevel}
		if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		if baoClient.loggerWriteCalls != 2 || baoClient.loggers[testLoggerName][testLoggerLevelField] != testLoggerLevel {
			t.Fatalf("writes = %d, logger = %#v, want drift corrected", baoClient.loggerWriteCalls, baoClient.loggers[testLoggerName])
		}
	})

	t.Run("refuses unexpected adoption", func(t *testing.T) {
		connection := readyConnection()
		logger := &openbaov1alpha1.OpenBaoLogger{
			ObjectMeta: metav1.ObjectMeta{Name: testLoggerResourceName, Namespace: testNamespace},
			Spec: openbaov1alpha1.OpenBaoLoggerSpec{
				ConnectionRef: connectionReference(connection),
				Name:          testLoggerName,
				Level:         testLoggerLevel,
			},
		}
		kubeClient := newTestClient(connection, logger)
		baoClient := &fakeSystemClient{loggers: map[string]openbaoclient.AdditionalObject{
			testLoggerName: {testLoggerLevelField: testLoggerInfoLevel},
		}}
		reconciler := newLoggerReconciler(kubeClient, baoClient)

		if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(logger)}); err == nil {
			t.Fatal("expected an adoption error")
		}
		var got openbaov1alpha1.OpenBaoLogger
		if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(logger), &got); err != nil {
			t.Fatal(err)
		}
		condition := findCondition(got.Status.Conditions)
		if condition == nil || condition.Reason != "OpenBaoAcquireFailed" || condition.Status != metav1.ConditionFalse {
			t.Fatalf("Ready condition = %#v, want OpenBaoAcquireFailed/False", condition)
		}
		if baoClient.loggerWriteCalls != 0 {
			t.Fatalf("writes = %d, want 0", baoClient.loggerWriteCalls)
		}
	})

	t.Run("adopts when explicitly requested", func(t *testing.T) {
		connection := readyConnection()
		logger := &openbaov1alpha1.OpenBaoLogger{
			ObjectMeta: metav1.ObjectMeta{Name: testLoggerResourceName, Namespace: testNamespace},
			Spec: openbaov1alpha1.OpenBaoLoggerSpec{
				ConnectionRef:  connectionReference(connection),
				Name:           testLoggerName,
				Level:          testLoggerInfoLevel,
				CreationPolicy: openbaov1alpha1.CreationPolicyAdopt,
			},
		}
		kubeClient := newTestClient(connection, logger)
		baoClient := &fakeSystemClient{loggers: map[string]openbaoclient.AdditionalObject{
			testLoggerName: {testLoggerLevelField: testLoggerInfoLevel},
		}}
		reconciler := newLoggerReconciler(kubeClient, baoClient)

		if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(logger)}); err != nil {
			t.Fatal(err)
		}
		if baoClient.loggerWriteCalls != 0 {
			t.Fatalf("writes = %d, want 0 for matching adoption", baoClient.loggerWriteCalls)
		}
		var got openbaov1alpha1.OpenBaoLogger
		if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(logger), &got); err != nil {
			t.Fatal(err)
		}
		if !conditionTrue(got.Status.Conditions) || got.Status.Level != testLoggerInfoLevel {
			t.Fatalf("status = %#v, want adopted logger Ready=True", got.Status)
		}
	})

	t.Run("orphan deletion never calls OpenBao", func(t *testing.T) {
		connection := readyConnection()
		deletionTime := metav1.NewTime(time.Now())
		logger := &openbaov1alpha1.OpenBaoLogger{
			ObjectMeta: metav1.ObjectMeta{
				Name:              testLoggerResourceName,
				Namespace:         testNamespace,
				Finalizers:        []string{finalizerName},
				DeletionTimestamp: &deletionTime,
			},
			Spec: openbaov1alpha1.OpenBaoLoggerSpec{
				ConnectionRef: connectionReference(connection), Name: testLoggerName, Level: testLoggerInfoLevel,
			},
		}
		kubeClient := newTestClient(connection, logger)
		baoClient := &fakeSystemClient{loggers: map[string]openbaoclient.AdditionalObject{
			testLoggerName: {"level": testLoggerInfoLevel},
		}}

		before := logger.Status.DeepCopy()
		if _, err := reconcileSystemNamed(context.Background(), kubeClient, nil, nil, systemPlan{
			Object:         logger,
			ConnectionRef:  connectionReference(connection),
			CreationPolicy: openbaov1alpha1.CreationPolicyCreate,
			DeletionPolicy: openbaov1alpha1.DeletionPolicyOrphan,
			Status:         oidcStatusView{Before: before, Current: &logger.Status, ConfigHash: &logger.Status.ConfigHash, ObservedGeneration: &logger.Status.ObservedGeneration, Conditions: &logger.Status.Conditions},
		}); err != nil {
			t.Fatal(err)
		}
		if baoClient.loggerDeleteCalls != 0 {
			t.Fatalf("delete calls = %d, want 0 for orphan policy", baoClient.loggerDeleteCalls)
		}
		if len(logger.Finalizers) != 0 {
			t.Fatalf("finalizers = %v, want none", logger.Finalizers)
		}
	})

	t.Run("retains delete finalizer when connection is unavailable", func(t *testing.T) {
		deletionTime := metav1.NewTime(time.Now())
		logger := &openbaov1alpha1.OpenBaoLogger{
			ObjectMeta: metav1.ObjectMeta{
				Name:              testLoggerResourceName,
				Namespace:         testNamespace,
				Finalizers:        []string{finalizerName},
				DeletionTimestamp: &deletionTime,
			},
			Spec: openbaov1alpha1.OpenBaoLoggerSpec{
				ConnectionRef:  openbaov1alpha1.OpenBaoConnectionReference{Name: testConnectionName},
				Name:           testLoggerName,
				Level:          testLoggerInfoLevel,
				DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
			},
		}
		kubeClient := newTestClient(logger)

		before := logger.Status.DeepCopy()
		result, err := reconcileSystemNamed(context.Background(), kubeClient, nil, nil, systemPlan{
			Object:         logger,
			ConnectionRef:  logger.Spec.ConnectionRef,
			CreationPolicy: openbaov1alpha1.CreationPolicyCreate,
			DeletionPolicy: openbaov1alpha1.DeletionPolicyDelete,
			Status:         oidcStatusView{Before: before, Current: &logger.Status, ConfigHash: &logger.Status.ConfigHash, ObservedGeneration: &logger.Status.ObservedGeneration, Conditions: &logger.Status.Conditions},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.RequeueAfter != dependencyRetry {
			t.Fatalf("result = %#v, want retry after %s", result, dependencyRetry)
		}
		assertCleanupRequired(t, logger.Status.Conditions)
		if len(logger.Finalizers) != 1 || logger.Finalizers[0] != finalizerName {
			t.Fatalf("finalizers = %v, want finalizer retained", logger.Finalizers)
		}
	})
}

func newLoggerReconciler(kubeClient client.Client, baoClient *fakeSystemClient) *OpenBaoLoggerReconciler {
	return &OpenBaoLoggerReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (SystemClient, error) {
			return baoClient, nil
		},
	}
}

func TestOIDCResourceAdaptersConformance(t *testing.T) {
	connection := readyConnection()
	cases := []struct {
		name      string
		object    client.Object
		reconcile func(client.Client, *fakeConformanceOIDCClient) error
	}{
		{
			name: testOIDCProviderName,
			object: &openbaov1alpha1.OpenBaoOIDCProvider{
				ObjectMeta: metav1.ObjectMeta{Name: testOIDCProviderName, Namespace: testNamespace},
				Spec:       openbaov1alpha1.OpenBaoOIDCProviderSpec{ConnectionRef: connectionReference(connection), Issuer: testOIDCIssuer, AllowedClientIDs: []string{"client-b", testOIDCClientName}},
			},
			reconcile: func(kubeClient client.Client, baoClient *fakeConformanceOIDCClient) error {
				r := &OpenBaoOIDCProviderReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
				_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: typesKey(testOIDCProviderName)})
				return err
			},
		},
		{
			name: testOIDCClientName,
			object: &openbaov1alpha1.OpenBaoOIDCClient{
				ObjectMeta: metav1.ObjectMeta{Name: testOIDCClientName, Namespace: testNamespace},
				Spec:       openbaov1alpha1.OpenBaoOIDCClientSpec{ConnectionRef: connectionReference(connection), ClientType: "confidential", RedirectURIs: []string{"https://b.example.test/cb", "https://a.example.test/cb"}},
			},
			reconcile: func(kubeClient client.Client, baoClient *fakeConformanceOIDCClient) error {
				r := &OpenBaoOIDCClientReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
				_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: typesKey(testOIDCClientName)})
				return err
			},
		},
		{
			name: "key",
			object: &openbaov1alpha1.OpenBaoOIDCKey{
				ObjectMeta: metav1.ObjectMeta{Name: "key", Namespace: testNamespace},
				Spec:       openbaov1alpha1.OpenBaoOIDCKeySpec{ConnectionRef: connectionReference(connection), Algorithm: "RS256", AllowedClientIDs: []string{testOIDCClientName}},
			},
			reconcile: func(kubeClient client.Client, baoClient *fakeConformanceOIDCClient) error {
				r := &OpenBaoOIDCKeyReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
				_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: typesKey("key")})
				return err
			},
		},
		{
			name: "role",
			object: &openbaov1alpha1.OpenBaoOIDCRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role", Namespace: testNamespace},
				Spec:       openbaov1alpha1.OpenBaoOIDCRoleSpec{ConnectionRef: connectionReference(connection), Key: "signing-key", ClientID: testOIDCClientName, Template: "{{identity.entity.name}}"},
			},
			reconcile: func(kubeClient client.Client, baoClient *fakeConformanceOIDCClient) error {
				r := &OpenBaoOIDCRoleReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
				_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: typesKey("role")})
				return err
			},
		},
		{
			name: "scope",
			object: &openbaov1alpha1.OpenBaoOIDCScope{
				ObjectMeta: metav1.ObjectMeta{Name: "scope", Namespace: testNamespace},
				Spec:       openbaov1alpha1.OpenBaoOIDCScopeSpec{ConnectionRef: connectionReference(connection), Description: "payments claims", Template: "{\"team\":\"payments\"}"},
			},
			reconcile: func(kubeClient client.Client, baoClient *fakeConformanceOIDCClient) error {
				r := &OpenBaoOIDCScopeReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
				_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: typesKey("scope")})
				return err
			},
		},
		{
			name: "assignment",
			object: &openbaov1alpha1.OpenBaoOIDCAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: "assignment", Namespace: testNamespace},
				Spec:       openbaov1alpha1.OpenBaoOIDCAssignmentSpec{ConnectionRef: connectionReference(connection), EntityIDs: []string{"entity-b", "entity-a"}, GroupIDs: []string{"group"}},
			},
			reconcile: func(kubeClient client.Client, baoClient *fakeConformanceOIDCClient) error {
				r := &OpenBaoOIDCAssignmentReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
				_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: typesKey("assignment")})
				return err
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			kubeClient := newTestClient(connection, testCase.object)
			baoClient := &fakeConformanceOIDCClient{}
			if err := testCase.reconcile(kubeClient, baoClient); err != nil {
				t.Fatal(err)
			}
			if baoClient.writeCalls != 1 {
				t.Fatalf("write calls = %d, want 1", baoClient.writeCalls)
			}
			resource := oidcResourceName(testCase.object)
			observed, ok := baoClient.resources[resource]
			if !ok || len(observed) == 0 {
				t.Fatalf("resource %q = %#v, want desired configuration", resource, observed)
			}
		})
	}
}

func TestOIDCSharedLifecycleCorrectsDriftAndControlsAdoption(t *testing.T) {
	connection := readyConnection()
	provider := &openbaov1alpha1.OpenBaoOIDCProvider{
		ObjectMeta: metav1.ObjectMeta{Name: testOIDCProviderName, Namespace: testNamespace},
		Spec:       openbaov1alpha1.OpenBaoOIDCProviderSpec{ConnectionRef: connectionReference(connection), Issuer: testOIDCIssuer},
	}
	request := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(provider)}

	t.Run("create and drift", func(t *testing.T) {
		kubeClient := newTestClient(connection, provider.DeepCopy())
		baoClient := &fakeConformanceOIDCClient{}
		reconciler := &OpenBaoOIDCProviderReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
		if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		baoClient.resources["provider/provider"] = openbaoclient.OIDCObject{testOIDCIssuerField: "https://drifted.example.test"}
		if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		if baoClient.writeCalls != 2 || baoClient.resources["provider/provider"][testOIDCIssuerField] != testOIDCIssuer {
			t.Fatalf("writes = %d, resource = %#v, want drift corrected", baoClient.writeCalls, baoClient.resources["provider/provider"])
		}
	})

	t.Run("refuses adoption by default", func(t *testing.T) {
		kubeClient := newTestClient(connection, provider.DeepCopy())
		baoClient := &fakeConformanceOIDCClient{resources: map[string]openbaoclient.OIDCObject{
			"provider/provider": {testOIDCIssuerField: testOIDCIssuer},
		}}
		reconciler := &OpenBaoOIDCProviderReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
		if _, err := reconciler.Reconcile(context.Background(), request); err == nil {
			t.Fatal("expected an adoption error")
		}
		if baoClient.writeCalls != 0 {
			t.Fatalf("write calls = %d, want 0", baoClient.writeCalls)
		}
	})
}

func TestOIDCConfigReconcilerCorrectsDrift(t *testing.T) {
	connection := readyConnection()
	object := &openbaov1alpha1.OpenBaoOIDCConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "issuer", Namespace: testNamespace},
		Spec:       openbaov1alpha1.OpenBaoOIDCConfigSpec{ConnectionRef: connectionReference(connection), Issuer: testOIDCIssuer},
	}
	kubeClient := newTestClient(connection, object)
	baoClient := &fakeConformanceOIDCClient{config: openbaoclient.OIDCObject{"issuer": "https://old.example.test"}}
	reconciler := &OpenBaoOIDCConfigReconciler{Client: kubeClient, NewClient: oidcTestClientFactory(baoClient)}
	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(object)}); err != nil {
		t.Fatal(err)
	}
	if baoClient.configWriteCalls != 1 || baoClient.config["issuer"] != object.Spec.Issuer {
		t.Fatalf("config writes = %d, config = %#v, want drift corrected", baoClient.configWriteCalls, baoClient.config)
	}
}

func TestSpecializedConfigurationNormalizesDesiredState(t *testing.T) {
	t.Run("token role sets and durations", func(t *testing.T) {
		role := &openbaov1alpha1.OpenBaoTokenRole{Spec: openbaov1alpha1.OpenBaoTokenRoleSpec{
			AllowedPolicies: []string{" z-policy ", "a-policy"},
			TokenBoundCIDRs: []string{"10.0.0.0/8", " 192.168.0.0/16"},
			TokenPeriod:     &metav1.Duration{Duration: time.Minute},
		}}
		request, err := desiredTokenRoleRequest(role)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(request.AllowedPolicies, []string{"a-policy", "z-policy"}) || !reflect.DeepEqual(request.TokenBoundCIDRs, []string{"10.0.0.0/8", "192.168.0.0/16"}) || request.TokenPeriod == nil || *request.TokenPeriod != 60 {
			t.Fatalf("request = %#v, want normalized sets and 60-second period", request)
		}
	})

	t.Run("rejects empty AppRole set values", func(t *testing.T) {
		role := &openbaov1alpha1.OpenBaoAppRole{Spec: openbaov1alpha1.OpenBaoAppRoleSpec{TokenPolicies: []string{testAliasEntityName, "  "}}}
		if _, err := desiredAppRoleRequest(role); err == nil {
			t.Fatal("expected empty AppRole policy value to be rejected")
		}
	})

	t.Run("preserves persona identity while comparing metadata", func(t *testing.T) {
		desired := openbaoclient.PersonaRequest{Name: testAliasEntityName, EntityID: testEntityID, MountAccessor: testPersonaMount, Metadata: map[string]string{testMetadataKey: testAliasEntityName}}
		observed := &openbaoclient.Persona{ID: "persona-1", Name: desired.Name, EntityID: desired.EntityID, MountAccessor: desired.MountAccessor, Metadata: map[string]string{testMetadataKey: testAliasEntityName}}
		if !personaMatches(desired, observed) {
			t.Fatal("persona with equivalent metadata should match")
		}
	})

	t.Run("password policy hash is content based", func(t *testing.T) {
		if passwordPolicyRulesHash("rules-a") == passwordPolicyRulesHash("rules-b") {
			t.Fatal("different password policies must have different hashes")
		}
		expectedHash := passwordPolicyRulesHash("rules-a")
		if passwordPolicyRulesHash("rules-a") != expectedHash {
			t.Fatal("same password policy must have a stable hash")
		}
	})
}

func TestPersonaReconcilerRejectsStableIdentityChange(t *testing.T) {
	connection := readyConnection()
	persona := &openbaov1alpha1.OpenBaoPersona{
		ObjectMeta: metav1.ObjectMeta{Name: "payments-persona", Namespace: testNamespace},
		Spec: openbaov1alpha1.OpenBaoPersonaSpec{
			ConnectionRef: connectionReference(connection),
			Name:          testAliasEntityName,
			EntityID:      testEntityID,
			MountAccessor: testPersonaMount,
		},
		Status: openbaov1alpha1.OpenBaoPersonaStatus{ID: "persona-1"},
	}
	kubeClient := newTestClient(connection, persona)
	baoClient := &fakePersonaClient{persona: &openbaoclient.Persona{
		ID: "persona-2", Name: testAliasEntityName, EntityID: testEntityID, MountAccessor: testPersonaMount,
	}}
	reconciler := &OpenBaoPersonaReconciler{
		Client: kubeClient,
		NewClient: func(context.Context, *openbaov1alpha1.OpenBaoConnection) (PersonaClient, error) {
			return baoClient, nil
		},
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(persona)}); err == nil {
		t.Fatal("expected persona identity mismatch")
	}
	var got openbaov1alpha1.OpenBaoPersona
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(persona), &got); err != nil {
		t.Fatal(err)
	}
	condition := findCondition(got.Status.Conditions)
	if condition == nil || condition.Reason != "PersonaIdentityMismatch" || condition.Status != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %#v, want PersonaIdentityMismatch/False", condition)
	}
}

func oidcTestClientFactory(baoClient *fakeConformanceOIDCClient) func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error) {
	return func(context.Context, *openbaov1alpha1.OpenBaoConnection) (OIDCClient, error) {
		return baoClient, nil
	}
}

func typesKey(name string) client.ObjectKey {
	return client.ObjectKey{Namespace: testNamespace, Name: name}
}

func oidcResourceName(object client.Object) string {
	switch object.(type) {
	case *openbaov1alpha1.OpenBaoOIDCProvider:
		return "provider/" + object.GetName()
	case *openbaov1alpha1.OpenBaoOIDCClient:
		return "client/" + object.GetName()
	case *openbaov1alpha1.OpenBaoOIDCKey:
		return "key/" + object.GetName()
	case *openbaov1alpha1.OpenBaoOIDCRole:
		return "role/" + object.GetName()
	case *openbaov1alpha1.OpenBaoOIDCScope:
		return "scope/" + object.GetName()
	case *openbaov1alpha1.OpenBaoOIDCAssignment:
		return "assignment/" + object.GetName()
	default:
		return ""
	}
}

type fakeConformanceOIDCClient struct {
	resources        map[string]openbaoclient.OIDCObject
	config           openbaoclient.OIDCObject
	writeCalls       int
	configWriteCalls int
}

func (f *fakeConformanceOIDCClient) GetOIDCResource(_ context.Context, resource, name string) (openbaoclient.OIDCObject, error) {
	value, ok := f.resources[resource+"/"+name]
	if !ok {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	return cloneOIDCObject(value), nil
}

func (f *fakeConformanceOIDCClient) WriteOIDCResource(_ context.Context, resource, name string, value openbaoclient.OIDCObject) error {
	if f.resources == nil {
		f.resources = map[string]openbaoclient.OIDCObject{}
	}
	f.writeCalls++
	f.resources[resource+"/"+name] = cloneOIDCObject(value)
	return nil
}

func (f *fakeConformanceOIDCClient) DeleteOIDCResource(_ context.Context, resource, name string) error {
	delete(f.resources, resource+"/"+name)
	return nil
}

func (f *fakeConformanceOIDCClient) GetOIDCConfig(context.Context) (openbaoclient.OIDCObject, error) {
	if f.config == nil {
		f.config = openbaoclient.OIDCObject{}
	}
	return cloneOIDCObject(f.config), nil
}

func (f *fakeConformanceOIDCClient) WriteOIDCConfig(_ context.Context, value openbaoclient.OIDCObject) error {
	f.configWriteCalls++
	f.config = cloneOIDCObject(value)
	return nil
}

func cloneOIDCObject(value openbaoclient.OIDCObject) openbaoclient.OIDCObject {
	result := make(openbaoclient.OIDCObject, len(value))
	maps.Copy(result, value)
	return result
}

type fakePersonaClient struct {
	persona *openbaoclient.Persona
}

func (f *fakePersonaClient) ListPersonaIDs(context.Context) ([]string, error) {
	if f.persona == nil {
		return nil, nil
	}
	return []string{f.persona.ID}, nil
}

func (f *fakePersonaClient) GetPersonaByID(context.Context, string) (*openbaoclient.Persona, error) {
	if f.persona == nil {
		return nil, &openbaoclient.HTTPError{StatusCode: 404}
	}
	copy := *f.persona
	return &copy, nil
}

func (f *fakePersonaClient) CreatePersona(context.Context, openbaoclient.PersonaRequest) (string, error) {
	return "persona-created", nil
}

func (f *fakePersonaClient) UpdatePersona(context.Context, string, openbaoclient.PersonaRequest) (*openbaoclient.Persona, error) {
	return f.persona, nil
}

func (f *fakePersonaClient) DeletePersona(context.Context, string) error { return nil }
