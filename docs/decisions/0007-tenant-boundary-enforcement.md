# Decision 0007: Enforce tenant boundaries at the Kubernetes and OpenBao layers

Status: accepted

## Context

OpenBao provides native multi-tenancy through namespaces and default-deny ACL
policies. Those controls constrain the token used for the OpenBao API, but they
cannot constrain the Kubernetes API access of the operator that holds that
token. Kubernetes namespace names also do not automatically create or select an
OpenBao namespace.

Harbor Operator demonstrates a useful deployment-level `watchNamespaces`
control, while Infisical Entity Operator adds explicit external organization
ownership. OpenBao Entity Operator already has same-namespace Kubernetes
references and immutable OpenBao namespace routing on `OpenBaoConnection`.

## Decision

The first tenant-boundary slice uses three platform-controlled layers:

1. `watchNamespaces` scopes the manager cache and the Helm chart's manager
   permissions to an explicit list of Kubernetes namespaces. The chart binds
   its generated manager `ClusterRole` with one `RoleBinding` per listed
   namespace and omits the manager `ClusterRoleBinding`. An empty list preserves
   the cluster-wide trusted-platform installation.
2. The selected `OpenBaoConnection` remains responsible for the external
   boundary. Platform administrators should provision an OpenBao namespace and
   least-privilege ACL policy there, then authenticate the operator with a
   token, AppRole, or Kubernetes Auth role that is limited to that namespace.
3. The chart publishes an unbound tenant-author ClusterRole that grants CRUD
   access to supported entity and durable configuration resources. Platform
   administrators bind it per tenant namespace; tenant principals do not receive access to
   OpenBaoConnection objects, connection subresources, or Secrets.

The operator will not add Kyverno as a runtime dependency, create an OpenBao
namespace lifecycle CRD, or add a first-class organization/tenant hierarchy in
this slice. Kubernetes RBAC and optional admission policy remain the platform's
controls for who may create connections, credentials, resource kinds, names,
and deletion policies.

## Consequences

- A separately installed operator can be limited to the Kubernetes namespaces
  and Secrets it must access, while OpenBao ACLs limit its external mutations.
- A shared trusted manager can serve multiple namespaces without making the
  platform-owned connections or credential material tenant-authorable.
- The default chart remains compatible with existing cluster-wide deployments.
- `watchNamespaces` is a runtime and RBAC boundary, not a complete hostile
  multi-tenancy model. A tenant must not be allowed to create arbitrary
  `OpenBaoConnection` objects or credential Secrets when the operator's
  external identity is platform-owned; use Kubernetes RBAC or admission policy
  for that authoring boundary.
- OpenBao namespaces can be configured through `OpenBaoNamespace`, but the
  OpenBao server and the platform's permission to create child namespaces
  remain external. Native OpenBao ACLs must constrain which namespace paths a
  tenant-authorized connection may mutate.
- The local Kind workflow verifies two tenant ServiceAccounts, two
  platform-owned connections, two OpenBao namespaces, and cross-boundary
  denial in both Kubernetes and OpenBao.
- A future fixed-connection or first-class OpenBao domain resource would need a
  new decision after a concrete use case demonstrates that platform RBAC and
  per-installation deployment scope are insufficient.
