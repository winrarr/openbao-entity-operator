# Tech-debt register

This register contains material limitations in the current implementation. Planned product outcomes belong in the [backlog](backlog.md); these entries describe known constraints or verification gaps.

## TD-001: Local live coverage uses OpenBao dev mode

Status: accepted limitation

The Kind workflow runs a single OpenBao 2.6.2 dev server with an in-memory data store and a generated local-only root token for bootstrap. It now also configures Kubernetes Auth and runs the operator with a non-root identity policy, but it does not prove persistence, HA/standby behavior, or production TLS configuration.

Exit criteria: add a separately provisioned integration environment that exercises a persistent, non-root authentication setup before making production deployment claims.

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

Status: accepted limitation

If a Delete-policy entity, alias, group, or ACL policy loses its `OpenBaoConnection` or credential Secret before its Kubernetes deletion is reconciled, the operator logs the loss and releases the finalizer to prevent a stuck Kubernetes object. This preserves cluster recoverability but cannot prove that the external object was deleted.

Exit criteria: introduce a recoverable connection and credential lifecycle that preserves cleanup access during dependent-resource deletion, then add live coverage proving external deletion remains possible after dependency ordering changes.
