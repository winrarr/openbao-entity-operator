# Architecture

## Components

```text
Kubernetes API
    │
    ├── OpenBaoConnectionReconciler ──┐
    │                                 │
    ├── OpenBaoEntityReconciler ──────┤
    ├── OpenBaoEntityAliasReconciler ─┤
    └── OpenBaoGroupReconciler ──────┼── internal/openbaoclient ── OpenBao HTTP API
                                      │
                              same-namespace Secret
```

The API types and Kubebuilder markers under `api/openbao/v1alpha1` are authoritative. CRDs and deepcopy methods are derived with `make manifests generate`. The OpenAPI file under `hack/` is external reference material only.

## Connection flow

`OpenBaoConnectionReconciler` reads the referenced Secret, constructs a client with the optional CA bundle and OpenBao namespace, and validates the token with `/v1/auth/token/lookup-self`. Root connections also call `/v1/sys/health`; health statuses such as sealed, standby, and uninitialized are decoded even when OpenBao reports them with non-2xx HTTP codes. OpenBao restricts `/sys/health` in child namespaces, so namespace-scoped connections establish readiness through namespaced token self-lookup. Tokens are never copied into status or log fields.

The controller watches referenced Secrets. Changes to a Secret enqueue only connections that reference it. Connections periodically recheck health and authentication so status can recover after OpenBao becomes available again.

## Entity flow

`OpenBaoEntityReconciler` resolves a ready connection before making external calls. It uses the recorded status ID as the stable binding after creation or adoption. If no ID exists, it looks up the entity by name and applies the creation policy. Once bound, it reads by ID, refuses an unexpected name mismatch, updates only when the desired state differs, and records the observed state in status.

`OpenBaoEntityAliasReconciler` resolves both the connection and a ready `OpenBaoEntity`, then binds the alias to the entity's current stable ID. OpenBao has no direct alias lookup by name and mount accessor, so initial adoption scans the alias ID list and reads candidates before applying the explicit creation policy. Once bound, the alias ID remains stable; external canonical-entity drift is corrected and changes to the referenced entity ID are propagated.

## Group and membership flow

`OpenBaoGroupReconciler` resolves a ready connection, acquires the group by its stable status ID or Kubernetes name, and reconciles metadata, policies, type, and external deletion using the same explicit ownership policies as entities. Internal groups can be targeted by `OpenBaoGroupMembership` resources. Each membership claims exactly one entity or subgroup and is watched through its parent group.

OpenBao exposes membership as arrays on the group update API rather than as independent membership endpoints. The controller therefore reads the current group, removes only membership IDs previously tracked as operator-managed, adds current claims, and writes the resulting arrays. Remote members that were never claimed remain intact. Membership status records the parent and member IDs; deleting a membership claim removes only that edge and never deletes the group or entity.

Deletion is safe by default: `Orphan` removes the Kubernetes finalizer without calling OpenBao. `Delete` adds a finalizer before external mutation and removes it only after the OpenBao object is deleted or already absent. If the connection or its credential Secret disappears first, the controller logs a warning and releases the finalizer so Kubernetes deletion cannot deadlock; the external object may remain and requires separate cleanup.

## Extension boundary

The typed client interfaces used by the reconcilers are intentionally narrow and injectable in tests. Future controllers can add OpenBao-native surfaces without turning the controller into a generic arbitrary-path reconciler. Namespace context is applied during centralized client construction, so every controller using a connection receives the same validated request boundary.
