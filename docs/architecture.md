# Architecture

## Components

```text
Kubernetes API
    │
    ├── OpenBaoConnectionReconciler ──┐
    │                                 │
    ├── OpenBaoPolicyReconciler ──────┤
    ├── OpenBaoEntityReconciler ──────┤
    ├── OpenBaoEntityAliasReconciler ─┤
    └── OpenBaoGroupReconciler ──────┼── internal/openbaoclient ── OpenBao HTTP API
                                      │
                  same-namespace Secret or projected ServiceAccount JWT
```

The API types and Kubebuilder markers under `api/openbao/v1alpha1` are authoritative. CRDs and deepcopy methods are derived with `make manifests generate`. The OpenAPI file under `hack/` is external reference material only.

The manager's optional `--watch-namespaces` setting is translated into
controller-runtime `DefaultNamespaces` cache configuration. The Helm chart
matches that runtime scope with namespace RoleBindings; an empty setting keeps
the existing cluster-wide manager ClusterRoleBinding. This controls Kubernetes
reads and watches only. The OpenBao client applies
`OpenBaoConnection.spec.namespace`, and the connection's ACL policy remains the
external authorization boundary.

The chart publishes two distinct RBAC surfaces. The manager permissions are
needed by reconcilers and should be bound only to the operator ServiceAccount.
The unbound tenant-author ClusterRole is a platform authoring profile for
tenant ServiceAccounts; it excludes connections, connection subresources,
Secrets, and manager finalizers. A platform-owned connection therefore remains
outside tenant mutation permissions while same-namespace references let tenant
resources use the connection by name.

## Connection flow

`OpenBaoConnectionReconciler` constructs a client with the configured authentication method, optional CA bundle, and OpenBao namespace. Token-authenticated connections read a same-namespace Secret and validate it with `/v1/auth/token/lookup-self`. Kubernetes-authenticated connections read the projected ServiceAccount JWT and log in through `/v1/auth/<mount>/login`; AppRole connections read same-namespace role ID and Secret ID sources and use the same login route shape. A manager-wide connection cache retains dynamic-auth clients so renewable leases can be renewed across reconciliations, and the client performs one fresh login retry after an authentication failure. Root connections also call `/v1/sys/health`; health statuses such as sealed, standby, and uninitialized are decoded even when OpenBao reports them with non-2xx HTTP codes. OpenBao restricts `/sys/health` in child namespaces, so namespace-scoped connections establish readiness through namespaced token self-lookup. Tokens, JWTs, and AppRole credentials are never copied into status or log fields.

The controller watches referenced Secrets when token authentication or a CA bundle is selected. Kubernetes-authenticated connections periodically re-read the projected JWT and recheck authentication so status can recover after OpenBao becomes available again.

## Entity flow

`OpenBaoEntityReconciler` resolves a ready connection before making external calls. It uses the recorded status ID as the stable binding after creation or adoption. If no ID exists, it looks up the entity by name and applies the creation policy. Once bound, it reads by ID, refuses an unexpected name mismatch, updates only when the desired state differs, and records the observed state in status.

`OpenBaoEntityAliasReconciler` resolves both the connection and a ready `OpenBaoEntity`, then binds the alias to the entity's current stable ID. OpenBao has no direct alias lookup by name and mount accessor, so initial adoption scans the alias ID list and reads candidates before applying the explicit creation policy. Once bound, the alias ID remains stable; external canonical-entity drift is corrected and changes to the referenced entity ID are propagated.

## Policy flow

`OpenBaoPolicyReconciler` uses the Kubernetes resource name as the OpenBao ACL
policy name and reconciles the exact raw document in `spec.rules` through
`/v1/sys/policies/acl/<name>`. A status name marks a policy already acquired by
the resource, so later reconciliations do not confuse its own managed policy
with an unrelated pre-existing policy. Initial existing policies require
`Adopt` or `CreateOrAdopt`; otherwise the controller reports a conflict without
overwriting the document. Status records a SHA-256 hash and OpenBao version,
while the document itself remains in spec and is not duplicated into status.

Policy deletion is orphaning by default and requires `deletionPolicy: Delete` to
call OpenBao. The controller watches its connection and uses the same shared
client/cache as the identity controllers, including Kubernetes Auth lease
handling and namespace routing.

## Group and membership flow

`OpenBaoGroupReconciler` resolves a ready connection, acquires the group by its stable status ID or Kubernetes name, and reconciles metadata, policies, type, and external deletion using the same explicit ownership policies as entities. Internal groups can be targeted by `OpenBaoGroupMembership` resources. Each membership claims exactly one entity or subgroup and is watched through its parent group.

OpenBao exposes membership as arrays on the group update API rather than as independent membership endpoints. The controller therefore reads the current group, removes only membership IDs previously tracked as operator-managed, adds current claims, and writes the resulting arrays. Remote members that were never claimed remain intact. Membership status records the parent and member IDs; deleting a membership claim removes only that edge and never deletes the group or entity.

Deletion is safe by default: `Orphan` removes the Kubernetes finalizer without calling OpenBao. `Delete` adds a finalizer before external mutation and removes it only after the OpenBao object is deleted or already absent. If the connection or its credential Secret disappears first, the controller retains the finalizer, records `CleanupRequired=True` and `Stalled=True`, and retries until the dependency is restored. This preserves recoverable external cleanup instead of silently orphaning the object; manual finalizer removal remains an administrative override.

## Extension boundary

The typed client interfaces used by the reconcilers are intentionally narrow and injectable in tests. Future controllers can add OpenBao-native surfaces without turning the controller into a generic arbitrary-path reconciler. Namespace context is applied during centralized client construction, so every controller using a connection receives the same validated request boundary.
