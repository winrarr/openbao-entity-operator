# Product scope

## Promise

OpenBao Entity Operator connects Kubernetes to an already deployed OpenBao
instance and reconciles durable OpenBao entities and API configuration as
Kubernetes custom resources. It is an entity operator in the sense that it
operates OpenBao entities and configuration after OpenBao exists; it is not an
OpenBao server operator.

The operator exposes readiness, observed identifiers, configuration hashes,
versions, and reconciliation failures through Kubernetes status. OpenBao ACLs,
OpenBao namespaces, Kubernetes RBAC, and installation scope remain the
authorization boundaries.

## Current scope

- Same-namespace `OpenBaoConnection` resources with an address, optional
  OpenBao namespace, exactly one of a token Secret, Kubernetes Auth, or AppRole
  credentials, an optional CA bundle Secret, and request timeout.
- ACL policy documents through `OpenBaoPolicy`.
- Identity entities, entity aliases, groups, group aliases, and explicit group
  membership claims.
- Identity personas, MFA methods, and MFA login-enforcement policies. MFA
  provider credentials are read from same-namespace Secrets and are never
  copied into status.
- Kubernetes Auth roles, token roles, and AppRole role configuration.
- Password policies.
- OIDC issuer configuration, providers, clients, keys, roles, scopes, and
  assignments. Generated OIDC client secrets are not copied into Kubernetes.
- Auth-method mounts and secret-engine mounts, including durable mount tuning.
- OpenBao namespaces, audit devices, rate-limit quotas, workflows, and plugin
  catalog registrations, plus CORS, audit request headers, UI headers, logger
  levels, global rate-limit settings, and automatic rotation configuration.
- Create, adopt, update, observe, drift-correct, orphan, and opt-in delete
  behavior where the OpenBao endpoint represents a durable named object.
- Helm installation for the controller and CRDs, with optional
  namespace-scoped installations. Kustomize remains an internal tool for
  sample rendering and generated artifacts.
- A latest-stable OpenBao support policy, currently OpenBao v2.6.2.

## Boundary with OpenBao deployment

OpenBao must be deployed and made reachable before the operator is installed.
The operator does not create or own an OpenBao `Deployment`, `StatefulSet`,
`Service`, storage volume, Raft cluster, TLS server configuration, or
OpenBao installation chart.

The following remain external deployment or operational workflows:

- initialization, seal configuration, unseal, recovery, rekey, root-token
  generation or rotation, and seal-status administration;
- storage, HA, Raft membership, backups, restores, upgrades, and server
  availability;
- installation or distribution of plugin binaries and external OCI content;
- issuance or synchronization of secret values, AppRole Secret IDs, OIDC
  client secrets, OpenBao tokens, and arbitrary secret-engine data;
- MFA credential generation or validation flows, including TOTP secret
  generation and one-shot MFA administration;
- immediate encryption-key or keyring rotation actions; the operator only
  manages the durable automatic-rotation configuration;
- one-shot actions such as lease revocation/tidy, step-down, remount, and OIDC
  authorize/token/introspection flows; and
- diagnostics and profiling endpoints.

These exclusions are about lifecycle semantics and credential safety, not a
product boundary around identity. Durable OpenBao API configuration belongs in
this operator when it has a stable desired state and an observable read/write
contract.

## Compatibility

The project targets the newest selected stable OpenBao release and does not
promise a backwards-compatibility matrix. Older versions may work when their
endpoint contracts are unchanged, but that is incidental. See the
[compatibility policy](compatibility.md).

## Security and tenancy

Credentials are read only from same-namespace Kubernetes Secrets or the
operator's projected ServiceAccount token. Credential values are never copied
to status, logs, samples, or generated fixtures. Native OpenBao permissions
are preferred for OpenBao-side boundaries; Kubernetes RBAC and scoped
installations constrain which Kubernetes resources and credentials the
operator can observe. See the [multi-tenancy guide](reference/multi-tenancy.md).
