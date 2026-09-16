# Quickstart

This operator manages an existing OpenBao instance. You need a reachable
OpenBao address, credentials with the required policy permissions, and cluster
access to install CRDs and the operator.

## Install the operator

For a published release, install the OCI chart:

```sh
helm upgrade --install openbao-entity-operator \
  oci://ghcr.io/winrarr/charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace \
  --version <chart-version>
```

For a checkout, use the chart directly:

```sh
helm upgrade --install openbao-entity-operator \
  charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace
```

See [Installation](introduction/installation.md) for chart values and
[local development](contributing/local-development.md) when working from a
checkout.

## Connect OpenBao

Create a Secret with a token that can manage the intended OpenBao resources,
then create a connection in the same namespace:

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

```sh
kubectl apply -f connection.yaml
kubectl wait --for=condition=Ready openbaoconnection/openbao --timeout=2m
```

For in-cluster Kubernetes Auth, use the [Kubernetes Auth operations
guide](operations/kubernetes-auth.md) instead of a static token Secret.

## Declare a policy and entity

The Kubernetes object name is the OpenBao policy or entity name:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  rules: |
    path "secret/data/payments/*" {
      capabilities = ["read"]
    }
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  policies:
    - payments
```

```sh
kubectl apply -f resources.yaml
kubectl get openbaopolicy,openbaoentity
kubectl get openbaopolicy/payments -o yaml
```

The `Ready` condition reports whether the desired external object is
reconciled. Policy status also reports the OpenBao version and a hash of the
observed policy document.

Continue with [Concepts](introduction/concepts.md), the [resource
guides](crds/index.md), and the [reference overview](reference/index.md).
