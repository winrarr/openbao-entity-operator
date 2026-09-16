# Tech-debt register

This register contains material limitations in the current implementation. Planned product outcomes belong in the [backlog](backlog.md); these entries describe known constraints or verification gaps.

## TD-001: Local live coverage uses OpenBao dev mode

Status: accepted limitation

The Kind workflow runs a single OpenBao 2.6.2 dev server with an in-memory data store and a generated local-only root token. It proves API and controller behavior, but not persistence, HA/standby behavior, TLS configuration, or production authentication policy.

Exit criteria: add a separately provisioned integration environment that exercises a persistent, non-root authentication setup before making production deployment claims.

## TD-002: The shared manager has cluster-wide Secret access

Status: accepted limitation

The manager watches namespaced resources cluster-wide and its generated ClusterRole can read Secrets in every namespace. Same-namespace references constrain the API model, but they do not provide tenant isolation between mutually untrusted users.

Exit criteria: add an independently scoped manager deployment and verify its cache, watch, and Secret permissions, or explicitly retain the trusted-platform deployment boundary as a product decision.

## TD-003: OpenBao API compatibility is snapshot-based

Status: open

The checked-in OpenAPI document is a runtime snapshot from OpenBao v2.6.2. OpenBao may change endpoint behavior or the generated document across releases, and the first slice has no version matrix.

Exit criteria: define the supported OpenBao version policy and run the live contract suite against every supported version before updating the snapshot or client behavior.

## TD-004: Delete-policy cleanup can orphan external objects after dependency loss

Status: accepted limitation

If a Delete-policy entity, alias, or group loses its `OpenBaoConnection` or credential Secret before its Kubernetes deletion is reconciled, the operator logs the loss and releases the finalizer to prevent a stuck Kubernetes object. This preserves cluster recoverability but cannot prove that the external object was deleted.

Exit criteria: introduce a recoverable connection and credential lifecycle that preserves cleanup access during dependent-resource deletion, then add live coverage proving external deletion remains possible after dependency ordering changes.
