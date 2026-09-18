# Backlog

These are real outcomes that are intentionally not part of the current supported slice.

## BL-001: Add explicit tenant-boundary controls

Status: complete for the supported platform-owned model

Goal: let platform operators deploy OpenBao Entity Operator with an explicit,
verifiable boundary for which Kubernetes namespaces, connections, and OpenBao
identity domains an installation may manage.

Rationale: the current project has useful same-namespace references and
optional OpenBao namespace routing, but the default manager watches the cluster
and can read Secrets in every namespace. Harbor Operator adds deployment-level
watch and reference controls, while Infisical Entity Operator models explicit
organization ownership for tenant principals. OpenBao Entity Operator does not
yet provide an equivalent boundary and therefore remains a trusted-platform
deployment.

Constraints: preserve the current safe same-namespace defaults; do not add
cross-namespace Secret access as a convenience feature; keep OpenBao namespace
lifecycle outside the operator unless a concrete design requires it; and make
the enforcement boundary observable in rendered RBAC, controller watches, and
documentation. The design may use separately scoped operator installations,
first-class connection/domain resources, runtime allowlists, or a deliberate
combination, but should not commit to one before the design is evaluated.

Acceptance criteria:

- A platform can restrict an installation to an explicit set of Kubernetes
  namespaces, and tests prove it does not watch or reconcile resources outside
  that set. **Completed for Helm installations with `watchNamespaces`.**
- A tenant cannot cause the operator to read another tenant's Secret or mutate
  an OpenBao connection or namespace outside the installation's declared
  boundary when the documented tenant-author RBAC profile is used. **Covered
  by the Kind multi-tenancy scenario.**
- Tenant-manageable resource kinds, naming rules, connection selection, and
  deletion blast radius are documented as enforceable Kubernetes policy, not
  implied by namespace names alone.
- A separately scoped installation can run with only the Kubernetes RBAC and
  OpenBao permissions needed for its tenant boundary. **Covered by scoped
  manager RoleBindings, the tenant-author ClusterRole, and the Kind
  multi-tenancy scenario.**
- Unit and controller tests cover boundary decisions and dependency rejection;
  focused live tests prove installation scope, tenant authoring permissions,
  and two permitted resource graphs without duplicating controller state-machine
  coverage. **Covered by the scope unit tests and Kind workflows.**

The supported boundary intentionally remains platform-configured. Do not add a
fixed operator-owned connection flag or a first-class OpenBao domain resource
until a concrete shared-installation use case shows that scoped installations,
platform-owned connections, Kubernetes RBAC, and OpenBao-native namespaces are
insufficient.

## BL-002: Manage Kubernetes Auth workload roles

Status: complete

Goal: let platform operators declare the ServiceAccount bindings and token
configuration of an OpenBao Kubernetes Auth role while keeping auth-mount
enablement and TokenReview administration outside this operator.

Rationale: workload authentication is a natural next resource after
Kubernetes-authenticated connections, but the operator should not become a
generic OpenBao configuration layer. A role-scoped CRD provides useful
declarative lifecycle management while OpenBao ACLs remain the native
authorization boundary.

Constraints: support only the newest selected OpenBao release; keep the
connection and mount path immutable; use explicit create/adopt and deletion
policies; do not store tokens or JWTs; compare sets deterministically; encode
durations as whole seconds; and keep live coverage focused on role lifecycle
and tenant isolation rather than duplicating OpenBao's own auth tests.

Acceptance criteria:

- `OpenBaoKubernetesAuthRole` manages bindings, token policies, and token
  lifetimes through the typed role endpoint with status hash and conditions.
- Existing roles require explicit adoption and external drift is corrected.
- Orphan and Delete semantics are covered, including retained finalizers when
  cleanup credentials are unavailable.
- The tenant-author RBAC profile includes the role resource without granting
  tenant access to platform-owned connections or Secrets.
- Unit, HTTP contract, generated-artifact, documentation, and focused Kind
  tests verify the feature.
