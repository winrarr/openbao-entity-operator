# Decision 0010: Manage Kubernetes Auth roles without managing the auth mount

Status: accepted

## Context

The operator already supports Kubernetes Auth as a connection credential, but
workloads also need declarative role bindings that connect Kubernetes
ServiceAccounts to OpenBao token policies. OpenBao exposes those bindings at
`auth/<mount>/role/<name>`. Enabling the mount and configuring its Kubernetes
API TokenReview credentials are platform administration concerns with a wider
blast radius than one workload role.

The operator's tenant model already uses same-namespace resources, platform-
owned connections, and native OpenBao ACLs. Adding an admission dependency or a
generic arbitrary-path resource would weaken that boundary and expand the test
surface unnecessarily.

## Decision

Add a namespaced `OpenBaoKubernetesAuthRole` resource with:

- an immutable same-namespace `connectionRef` and `mountPath`;
- explicit create, adopt, create-or-adopt, orphan, and delete semantics;
- required ServiceAccount name and namespace sets;
- token policy and whole-second token lifetime fields; and
- status containing only the mount, role name, normalized configuration hash,
  observed generation, and conditions.

The controller manages only the role endpoint. The selected connection's
OpenBao credential must authorize that endpoint, and the auth mount,
TokenReview configuration, and cluster trust remain outside the operator.
Tenant author RBAC includes the role CRD, while connections and Secrets remain
platform-owned. The default role deletion policy is Orphan.

## Consequences

- Workload role intent can be reviewed and drift-corrected through Kubernetes.
- Native OpenBao ACLs enforce the external tenant boundary without a Kyverno
  runtime dependency.
- A platform must still bootstrap and operate the Kubernetes Auth mount before
  a role resource can become Ready.
- Future auth-method configuration resources would need a separate design and
  should not be smuggled into this role API.
