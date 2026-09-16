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
| Operator scope | The default manager watches the cluster and can read Secrets in every namespace | This is a trusted platform boundary, not tenant isolation |

An OpenBao namespace is an external OpenBao isolation boundary. It does not
automatically map to a Kubernetes namespace, restrict who may submit a CR, or
limit the manager's Kubernetes Secret access.

## What is not supported yet

The operator does not currently provide Harbor Operator-style runtime controls
such as a fixed connection selected for the whole deployment, an explicit
`watchNamespaces` allowlist, or a configurable cross-namespace reference
policy. It also does not provide Infisical Entity Operator-style organization
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
tenant boundary. Adding first-class scoped deployment controls is tracked in
the [backlog](../backlog.md).

The comparison that informed this boundary is recorded in the
[multi-tenancy research note](../research/2026-09-16-multi-tenancy.md).
