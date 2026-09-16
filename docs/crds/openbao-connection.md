# OpenBaoConnection

`OpenBaoConnection` defines how the operator reaches one OpenBao API boundary.
It is namespaced, and every dependent resource must reference a connection in
the same namespace.

## Token authentication

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openbao-token
type: Opaque
stringData:
  token: ${OPENBAO_TOKEN}
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao
spec:
  address: https://openbao.example.com:8200
  tokenSecretRef:
    name: openbao-token
    key: token
```

The Secret must be in the same Kubernetes namespace as the connection. The
operator validates the token with OpenBao and never copies it into status.

## Kubernetes Auth

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao-kubernetes-auth
spec:
  address: https://openbao.example.com:8200
  kubernetesAuth:
    mountPath: kubernetes
    role: openbao-entity-operator
```

OpenBao must already have the Kubernetes Auth method and role configured. The
chart's ServiceAccount token automounting must remain enabled. See the
[Kubernetes Auth guide](../operations/kubernetes-auth.md) for the OpenBao-side
setup.

## Namespace and CA boundaries

Set `spec.namespace` to route requests to an OpenBao namespace. The value is
sent as the `X-Vault-Namespace` request header; an empty value targets the root
namespace. A private OpenBao CA can be supplied with `caBundleSecretRef`.

The connection reference, namespace, authentication method, and relevant
credential references are immutable. Update dependent resources only after the
connection is `Ready=True`.

## Status and failure behavior

The `Ready` condition reflects authentication and connectivity. Root namespace
connections also report OpenBao health details; namespaced connections use
token self-lookup because OpenBao restricts the root health endpoint there.

Inspect the resource and controller logs when a dependent object remains
blocked:

```sh
kubectl get openbaoconnection/openbao -o yaml
kubectl describe openbaoconnection/openbao
kubectl logs deployment/openbao-entity-operator \
  -n openbao-entity-operator-system
```

See the [generated OpenBaoConnection schema](../reference/api.md#openbaoconnection)
for the complete field reference.
