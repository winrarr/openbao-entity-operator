# Architecture

## Components

```text
Kubernetes API
    │
    ├── OpenBaoConnectionReconciler ──┐
    │                                 │
    ├── OpenBaoEntityReconciler ──────┤
    └── OpenBaoEntityAliasReconciler ─┼── internal/openbaoclient ── OpenBao HTTP API
                                      │
                              same-namespace Secret
```

The API types and Kubebuilder markers under `api/openbao/v1alpha1` are authoritative. CRDs and deepcopy methods are derived with `make manifests generate`. The OpenAPI file under `hack/` is external reference material only.

## Connection flow

`OpenBaoConnectionReconciler` reads the referenced Secret, constructs a client with the optional CA bundle, calls `/v1/sys/health`, and validates the token with `/v1/auth/token/lookup-self`. Health statuses such as sealed, standby, and uninitialized are decoded even when OpenBao reports them with non-2xx HTTP codes. Tokens are never copied into status or log fields.

The controller watches referenced Secrets. Changes to a Secret enqueue only connections that reference it. Connections periodically recheck health and authentication so status can recover after OpenBao becomes available again.

## Entity flow

`OpenBaoEntityReconciler` resolves a ready connection before making external calls. It uses the recorded status ID as the stable binding after creation or adoption. If no ID exists, it looks up the entity by name and applies the creation policy. Once bound, it reads by ID, refuses an unexpected name mismatch, updates only when the desired state differs, and records the observed state in status.

`OpenBaoEntityAliasReconciler` resolves both the connection and a ready `OpenBaoEntity`, then binds the alias to the entity's current stable ID. OpenBao has no direct alias lookup by name and mount accessor, so initial adoption scans the alias ID list and reads candidates before applying the explicit creation policy. Once bound, the alias ID remains stable; external canonical-entity drift is corrected and changes to the referenced entity ID are propagated.

Deletion is safe by default: `Orphan` removes the Kubernetes finalizer without calling OpenBao. `Delete` adds a finalizer before external mutation and removes it only after the OpenBao entity is deleted or already absent.

## Extension boundary

The typed client interfaces used by the reconcilers are intentionally narrow and injectable in tests. Future controllers can add OpenBao-native surfaces without turning the controller into a generic arbitrary-path reconciler. Groups should bind to explicit entity or connection references and must preserve stable external IDs before they are added.
