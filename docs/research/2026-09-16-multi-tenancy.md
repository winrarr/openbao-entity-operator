# Multi-tenancy comparison — 2026-09-16

## Inputs

The comparison used the local working copies of the related projects and their
durable documentation:

- [Harbor Operator multi-tenancy guide](https://github.com/winrarr/harbor-operator/blob/main/docs/reference/multi-tenancy.md)
- [Harbor Operator concepts](https://github.com/winrarr/harbor-operator/blob/main/docs/introduction/concepts.md)
- [Infisical Entity Operator security guide](https://github.com/winrarr/infisical-entity-operator/blob/main/docs/reference/security.md)
- [Infisical organization-scoped identity decision](https://github.com/winrarr/infisical-entity-operator/blob/main/docs/decisions/0006-organization-scoped-identities-for-tenant-principals.md)
- [Infisical organization tenant-boundary decision](https://github.com/winrarr/infisical-entity-operator/blob/main/docs/decisions/0007-explicit-organization-tenant-boundaries.md)

The OpenBao comparison used the current API types, generated RBAC, controller
reference resolution, and manager configuration in this repository.

## Observed patterns

Harbor Operator separates namespaced tenant-local connections from a
cluster-scoped shared connection. It also exposes deployment controls for the
watched namespace set and cross-namespace references, while recommending
admission policy for tenant naming, resource-kind allowlists, and deletion
blast-radius controls.

Infisical Entity Operator models a tenant hierarchy explicitly with
`InfisicalOrganization` and organization/project references. Dependent
resources validate the observed organization ID, and its security guide
describes organization-scoped machine identities as the external tenant
boundary. Kubernetes RBAC, admission, or per-vCluster deployment remains part
of the enforcement model.

OpenBao Entity Operator currently has namespaced CRDs, same-namespace object and
Secret references, and immutable OpenBao namespace routing on
`OpenBaoConnection`. It has no cluster-scoped connection, watched-namespace
allowlist, cross-namespace reference setting, or OpenBao namespace lifecycle
resource. The default manager ClusterRole can read Secrets cluster-wide, so the
current deployment is intentionally a trusted-platform model.

## Conclusion

The project has useful namespace-local safety properties and can target an
already provisioned OpenBao namespace, but it does not yet support the full
operator-level or domain-level tenancy capabilities represented by the two
related projects. A future design should decide whether OpenBao tenancy is
best expressed through scoped operator deployments, first-class connection and
namespace resources, or both. That decision is intentionally left to the
multi-tenancy backlog item rather than inferred into the current API.
