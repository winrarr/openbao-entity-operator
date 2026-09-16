# OpenBao Kubernetes Auth

`OpenBaoConnection` supports either a same-namespace token Secret or OpenBao's
Kubernetes Auth method. Kubernetes Auth is useful when the operator should not
hold a long-lived OpenBao token Secret.

## OpenBao-side setup

An OpenBao administrator must enable and configure the Kubernetes Auth method,
then create a role bound to the operator's Kubernetes ServiceAccount and
namespace. The role policy should grant only the identity and token operations
needed by the resources managed through that connection. The operator does not
enable auth mounts, configure TokenReview credentials, create policies, or
create roles.

The auth mount defaults to `kubernetes`. Set `spec.kubernetesAuth.mountPath`
when OpenBao uses another mount path. The value is the mount path below `auth/`,
not a complete `auth/...` URL.

## Kubernetes-side setup

The operator chart enables ServiceAccount token automounting by default. Keep
it enabled for Kubernetes-authenticated connections. If the chart uses an
existing ServiceAccount, that ServiceAccount must also provide a projected token
to the controller pod.

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao
  namespace: platform
spec:
  address: https://openbao.example.com:8200
  kubernetesAuth:
    role: openbao-entity-operator
  caBundleSecretRef:
    name: openbao-ca
    key: ca.crt
```

`tokenSecretRef` and `kubernetesAuth` are mutually exclusive and exactly one
must be configured. `caBundleSecretRef` is optional and can be used with either
authentication method for a private OpenBao CA.

## Token lifecycle

The controller reads the projected JWT only when it needs to log in. The manager
keeps the client and returned OpenBao token in memory for each connection, renews
renewable leases before they expire, and logs in again if renewal fails or
OpenBao returns an authentication failure. A changed role, mount, address,
namespace, timeout, or CA bundle creates a new client. Credential material is
never copied into resource status or log fields.

If the auth role, ServiceAccount binding, TokenReview configuration, or projected
token is invalid, the connection remains not ready and dependent identity
resources do not perform OpenBao mutations.

See the [local Kind guide](local-kind.md) for a complete disposable setup and
the [OpenBao API research note](../research/2026-09-16-openbao-api.md) for the
versioned API evidence behind this flow.
