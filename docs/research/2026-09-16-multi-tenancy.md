# Multi-tenancy comparison — 2026-09-16

## Inputs

The comparison used the local working copies of the related projects and their
durable documentation:

- [Harbor Operator multi-tenancy guide](https://github.com/winrarr/harbor-operator/blob/main/docs/reference/multi-tenancy.md)
- [Harbor Operator concepts](https://github.com/winrarr/harbor-operator/blob/main/docs/introduction/concepts.md)
- [Infisical Entity Operator security guide](https://github.com/winrarr/infisical-entity-operator/blob/main/docs/reference/security.md)
- [Infisical organization-scoped identity decision](https://github.com/winrarr/infisical-entity-operator/blob/main/docs/decisions/0006-organization-scoped-identities-for-tenant-principals.md)
- [Infisical organization tenant-boundary decision](https://github.com/winrarr/infisical-entity-operator/blob/main/docs/decisions/0007-explicit-organization-tenant-boundaries.md)
- [OpenBao namespaces](https://openbao.org/docs/next/concepts/namespaces/)
- [OpenBao security model](https://openbao.org/docs/internals/security/)
- [OpenBao Kubernetes Auth](https://openbao.org/docs/next/auth/kubernetes/)

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
`OpenBaoConnection`. It has no cluster-scoped connection, cross-namespace
reference setting, or OpenBao namespace lifecycle resource. The default manager
ClusterRole can read Secrets cluster-wide, so an empty-scope deployment remains
a trusted-platform model. A Helm installation can now set `watchNamespaces` to
limit both the controller-runtime cache and the manager RoleBindings to an
explicit Kubernetes namespace set.

OpenBao namespaces are the stronger external boundary: they isolate policies,
auth methods, entities, groups, tokens, and secret engines, while ACL policies
default to deny unless a token's associated policies grant a capability. The
Kubernetes Auth role can additionally bind authentication to specific
Kubernetes ServiceAccount names and namespaces. These controls are native to
OpenBao and should be configured on the connection credential rather than
reimplemented as operator tenant objects.

## Conclusion

The project has useful namespace-local safety properties and can target an
already provisioned OpenBao namespace. The accepted first slice is scoped
operator deployment plus native OpenBao ACL/namespace enforcement; it does not
attempt to reproduce Infisical's organization hierarchy inside Kubernetes.
Kyverno remains an optional admission-policy tool for authoring constraints, not
a replacement for OpenBao ACLs or Kubernetes RBAC. A fixed connection or
first-class OpenBao domain resource remains deferred until a concrete shared
installation needs stronger connection ownership than platform RBAC provides.
