# Multi-tenancy

OpenBao Entity Operator currently supports a namespace-local, trusted-platform
operating model. It is not yet a complete tenant-isolation solution for
mutually untrusted Kubernetes users.

## What is supported today

| Boundary | Current behavior | What it provides |
| --- | --- | --- |
| Kubernetes custom resources | All operator resources are namespaced | Kubernetes RBAC can limit which namespaces a tenant may manage |
| Resource references | All connection, policy, entity, alias, group, membership, and credential references are same-namespace | A CR cannot select another namespace's referenced object through the API |
| OpenBao namespace | `OpenBaoConnection.spec.namespace` routes every request through one OpenBao namespace | External identity and policy isolation when OpenBao namespaces are already provisioned and authorized |
| OpenBao connection sharing | Each `OpenBaoConnection` is namespaced; there is no cluster-scoped connection | A platform can create one connection per tenant namespace, but there is no built-in shared-connection policy |
| Operator scope | `watchNamespaces` can limit the manager cache and Helm manager permissions to listed namespaces; empty keeps cluster-wide behavior | A separately scoped installation can avoid reading or watching other namespaces |

An OpenBao namespace is an external OpenBao isolation boundary. It does not
automatically map to a Kubernetes namespace, restrict who may submit a CR, or
limit the manager's Kubernetes Secret access.

## Supported platform-owned tenant boundary

The supported shared-cluster model is a trusted platform manager with tenant
authoring constrained by Kubernetes RBAC and each tenant's external identity
constrained by an OpenBao namespace:

1. Set `watchNamespaces` to the exact Kubernetes namespaces the installation
   serves. The chart creates manager RoleBindings only in those namespaces.
2. Provision one OpenBao namespace and one least-privilege Kubernetes Auth role
   per tenant. OpenBao namespaces isolate policies, auth methods, entities,
   groups, tokens, and other identity state; see the [OpenBao namespace
   documentation](https://openbao.org/docs/next/concepts/namespaces/).
3. Create one platform-owned `OpenBaoConnection` in each tenant namespace. Set
   its immutable `spec.namespace` and Kubernetes Auth role to the matching
   OpenBao namespace. Prefer Kubernetes Auth so no static credential Secret is
   needed.
4. Bind `<release-name>-tenant-author-role` to the tenant ServiceAccount with a
   namespace `RoleBinding`. This profile allows CRUD access to policies,
   entities, aliases, groups, and group memberships only. It does not grant
   access to connections or Secrets.
5. Use admission policy when the platform needs additional rules for names,
   deletion policies, labels, or resource kinds. The operator intentionally
   does not become a tenant admission controller.

This model gives tenants separate Kubernetes authoring scopes and separate
OpenBao identity domains. A tenant can know the name of its platform-owned
connection and reference it from its own resources, but cannot read or modify
the connection or its credential material. Because references are same-
namespace, it cannot name the other tenant's connection from its own namespace.

The local Kind workflow exercises this complete boundary with two Kubernetes
namespaces, two OpenBao namespaces, two tenant ServiceAccounts, and real
Kubernetes-authenticated OpenBao tokens. It checks both directions of
Kubernetes API access and both directions of OpenBao namespace mutation.

## Native OpenBao boundary

For a separately scoped installation, provision the external boundary in
OpenBao and give the connection only the permissions needed by that tenant. A
typical Kubernetes Auth setup is conceptually:

```sh
bao namespace create team-a

BAO_NAMESPACE=team-a bao policy write team-a-operator - <<'EOF'
path "auth/token/lookup-self" {
  capabilities = ["read"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}

path "identity/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "sys/policies/acl/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF

BAO_NAMESPACE=team-a bao write auth/kubernetes/role/openbao-entity-operator \
  bound_service_account_names=openbao-entity-operator \
  bound_service_account_namespaces=openbao-entity-operator-system \
  token_policies=team-a-operator
```

The matching `OpenBaoConnection` sets `spec.namespace: team-a` and selects the
Kubernetes Auth role. The policy paths are relative to that OpenBao namespace;
the token cannot use them to reach another namespace, and the operator does not
need permission to create or administer namespaces. Tailor the policy to the
resource kinds enabled for the installation rather than copying this broad
identity example unchanged.

## What is not supported yet

The operator does not currently provide Harbor Operator-style runtime controls
such as a fixed shared connection selected for the whole deployment or a
configurable cross-namespace reference policy. It provides the narrower
`watchNamespaces` deployment scope plus a reusable platform-owned tenant RBAC
profile. It also does not provide Infisical Entity Operator-style organization
and project CRDs with organization IDs validated on dependent resources.

Consequently, the operator does not claim hostile multi-tenancy. A shared
installation should use Kubernetes RBAC and admission policy to control at
least:

- which namespaces and resource kinds each tenant may submit;
- tenant-specific names for OpenBao-global entities, groups, and policies;
- which `OpenBaoConnection` and OpenBao namespace each tenant may select; and
- whether `deletionPolicy: Delete` is permitted for tenant-managed resources.

For stronger isolation, deploy separately scoped operator instances and give
each instance only the Kubernetes permissions and OpenBao credentials for its
tenant boundary. With Helm, `watchNamespaces` creates namespace RoleBindings
instead of the manager ClusterRoleBinding. Configure the selected connection's
OpenBao namespace and ACL policy natively in OpenBao; do not use a root token or
allow tenant authors to create arbitrary platform-owned credential connections.
The tenant-author role is a permission profile and does not prevent a separate
RoleBinding from accidentally granting broader access; review effective RBAC
and admission rules as part of platform setup.

This is not a hostile multi-tenancy claim by itself. Tenant authoring rules
still belong in Kubernetes RBAC and, where needed, admission policy such as
Kyverno. Kyverno is not required for the runtime namespace boundary, and the
operator does not depend on it.

The platform-owned connection and tenant resource-kind model is recorded in
[Decision 0007](../decisions/0007-tenant-boundary-enforcement.md). A future
fixed-connection or first-class domain resource remains deliberately deferred
until a concrete shared-installation use case requires controller-enforced
selection.

The comparison that informed this boundary is recorded in the
[multi-tenancy research note](../research/2026-09-16-multi-tenancy.md).
