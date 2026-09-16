# Connection patterns

`OpenBaoConnection` is the boundary for address, authentication, namespace,
and TLS configuration. Resource references are same-namespace by design.

## Static token Secret

Use a same-namespace Secret when an existing OpenBao token is the simplest
bootstrap or operational choice:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao
  namespace: platform
spec:
  address: https://openbao.example.com:8200
  tokenSecretRef:
    name: openbao-token
    key: token
```

Grant the token only the OpenBao capabilities needed by the resources in that
namespace. Do not place the token in a manifest committed to the repository.

## Kubernetes Auth

Use Kubernetes Auth when the operator should authenticate with its projected
ServiceAccount JWT rather than a long-lived Kubernetes Secret:

```yaml
spec:
  address: https://openbao.example.com:8200
  kubernetesAuth:
    mountPath: kubernetes
    role: openbao-entity-operator
```

The OpenBao administrator must configure the auth method, TokenReview access,
role bindings, and policy. The [Kubernetes Auth guide](../operations/kubernetes-auth.md)
covers the setup and token lifecycle.

## OpenBao namespaces

Set `spec.namespace` to route requests to a namespace in an OpenBao Enterprise
or compatible namespace deployment. The value is absolute or relative to the
configured OpenBao namespace context. It is immutable and applies to all
resources using the connection.

Root connections use `/v1/sys/health` for health details. Namespace-scoped
connections establish readiness with token self-lookup because root health
fields are not available from child namespaces.

## Private CA material

Use `caBundleSecretRef` with either authentication method:

```yaml
spec:
  address: https://openbao.internal.example.com:8200
  caBundleSecretRef:
    name: openbao-ca
    key: ca.crt
  tokenSecretRef:
    name: openbao-token
    key: token
```

The CA Secret is read from the same Kubernetes namespace and is watched for
changes. A changed address, namespace, authentication mode, role, timeout, or
CA bundle causes the connection client to be rebuilt.
