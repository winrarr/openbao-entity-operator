# Tech-debt register

This register contains material limitations in the current implementation. Planned product outcomes belong in the [backlog](backlog.md); these entries describe known constraints or verification gaps.

## TD-001: Default live coverage uses OpenBao dev mode

Status: partially addressed; operator resilience covered

The default Kind workflow runs a single OpenBao 2.6.2 dev server with an in-memory data store and a generated local-only root token for bootstrap. The opt-in resilience workflow adds a separately provisioned one-node persistent/TLS OpenBao fixture, a non-root operator token, a server Pod restart, and token revocation/rotation recovery. It deliberately treats storage, Raft, initialization, unseal, TLS, ACL semantics, HA/standby behavior, backups, and restore as fixture plumbing rather than product assertions.

Exit criteria: add a separately provisioned integration environment only if the project needs production claims about OpenBao deployment, HA, storage, backup, or restore behavior. The current resilience profile is sufficient for the operator's reconnect and credential-rotation claims.

## TD-002: The shared manager has cluster-wide Secret access

Status: partially addressed; scoped model covered

The default manager watches namespaced resources cluster-wide and its generated
ClusterRole can read Secrets in every namespace. Helm installations can now set
`watchNamespaces`, which scopes the cache and replaces the manager
ClusterRoleBinding with namespace RoleBindings. The chart also provides an
unbound tenant-author ClusterRole that omits connections and Secrets; the
platform must bind it instead of the generated CRD editor/admin roles. This
scoped model still does not provide complete tenant isolation if a platform
grants tenants broader Kubernetes RBAC or uses a shared root OpenBao credential.

Exit criteria: keep the documented scoped, platform-owned model covered by the
Kind isolation scenario; add a fixed connection or first-class domain resource
only if a concrete shared-installation use case requires controller-enforced
connection selection. See the
[multi-tenancy guide](reference/multi-tenancy.md).

## TD-003: OpenBao API compatibility is snapshot-based

Status: accepted policy

The checked-in OpenAPI document is a runtime snapshot from the newest stable
OpenBao release selected for the project line, currently v2.6.2. The project
does not maintain a backwards-compatibility matrix or promise support for
older releases and pre-release builds.

Exit criteria: reconsider only if a concrete supported deployment requires an
older release or if the latest-stable policy becomes impractical. See the
[compatibility policy](compatibility.md).

## TD-004: Delete-policy cleanup can orphan external objects after dependency loss

Status: complete

Delete-policy resources now retain their finalizer when the referenced
`OpenBaoConnection`, credential Secret, or external deletion request is
unavailable. They report `CleanupRequired=True` and retry until cleanup can be
proven. Unit tests cover connection loss, credential loss, and recovery, and
the Kind workflow covers the live credential-loss path.

The accepted recovery trade-off is that a resource can remain `Terminating`
until the platform restores access. See [Decision 0008](decisions/0008-deletion-finalizer-retention.md).

## TD-005: Broad controller coverage is representative, not exhaustive

Status: addressed as the accepted v0.2 test boundary

The operator has more resource kinds than can be usefully exercised by a
small live test suite. The repository now centralizes shared lifecycle
conformance tests, covers all OIDC adapters through a table-driven fixture,
checks specialized normalization and identity invariants, and renders grouped
samples. Individual endpoint behavior remains covered by typed HTTP contract
tests; Kind remains focused on installation and external wiring.

Exit criteria: retain the `make conformance` and `make samples-check` targets,
add focused unit or HTTP coverage when a new endpoint or lifecycle rule is
introduced, and add a live scenario only when fake clients cannot observe the
behavior. Do not convert this register entry into a requirement for one E2E
scenario per CRD.
