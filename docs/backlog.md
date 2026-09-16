# Backlog

These are real outcomes that are intentionally not part of the current supported slice.

## BL-001: Add explicit tenant-boundary controls

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
  that set.
- A tenant cannot cause the operator to read another tenant's Secret or mutate
  an OpenBao connection or namespace outside the installation's declared
  boundary.
- Tenant-manageable resource kinds, naming rules, connection selection, and
  deletion blast radius are documented as enforceable Kubernetes policy, not
  implied by namespace names alone.
- A separately scoped installation can run with only the Kubernetes RBAC and
  OpenBao permissions needed for its tenant boundary.
- Unit and controller tests cover boundary decisions and dependency rejection;
  a focused live test proves installation scope and one permitted resource
  graph without duplicating controller state-machine coverage.
