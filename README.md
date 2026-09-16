# OpenBao Entity Operator

Kubernetes-native lifecycle management for OpenBao identity entities and groups.

The current vertical slice gives platform teams a declarative boundary around one OpenBao instance and its identity resources:

```text
Kubernetes Secret → OpenBaoConnection → OpenBaoEntity / OpenBaoGroup → membership claims
```

The operator validates connectivity, reconciles entity metadata, policies, and disabled state, binds auth-method aliases to entities, manages internal groups and explicit membership edges, reports stable external IDs in status, detects drift, and makes external deletion an explicit choice. It is OpenBao-focused; Vault compatibility is not a project promise.

## Quick start

Create a Secret containing an OpenBao token, then apply a connection and entity in the same namespace:

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
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  metadata:
    owner: platform
  policies:
    - default
    - payments
  deletionPolicy: Orphan
```

The entity's Kubernetes `metadata.name` is its OpenBao name. Use `creationPolicy: Adopt` or `CreateOrAdopt` when an entity already exists and should be managed instead of treated as a conflict. `deletionPolicy: Delete` is opt-in and permanently removes the recorded OpenBao entity.

To target an isolated OpenBao namespace, add `namespace: platform/production` to the connection. The namespace is immutable; create a new connection when moving resources to another OpenBao namespace. Namespace-scoped connections validate the token in that namespace and leave root-only health fields unpopulated.

For an in-cluster deployment, the connection can use OpenBao's Kubernetes Auth
method instead of a static token Secret. The auth mount and role must already be
configured in OpenBao, and the chart's ServiceAccount token automount must stay
enabled:

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

The client renews renewable auth tokens and performs one fresh login retry after
OpenBao rejects a token. See the [Kubernetes Auth operations guide](docs/operations/kubernetes-auth.md)
for the OpenBao-side setup and rotation considerations.

An `OpenBaoEntityAlias` references the entity resource, an OpenBao auth-method mount accessor, and the alias name presented by that auth method. Its `status.canonicalID` records the bound entity ID; aliases default to safe orphaning and can opt into external deletion.

An `OpenBaoGroup` uses its Kubernetes name as the OpenBao group name. Create an `OpenBaoGroupMembership` for each entity or subgroup that should be claimed by the group:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroup
metadata:
  name: platform
spec:
  connectionRef:
    name: openbao
  policies:
    - default
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupMembership
metadata:
  name: platform-payments
spec:
  groupRef:
    name: platform
  entityRef:
    name: payments
```

Membership resources manage only their claimed relationship. Existing remote memberships that are not claimed remain untouched, and deleting a membership claim removes its relationship without deleting the group or entity.

## Install

### Published chart

Once a release is published, install the OCI chart with:

```sh
helm upgrade --install openbao-entity-operator \
  oci://ghcr.io/winrarr/charts/openbao-entity-operator \
  --version <chart-version> \
  --namespace openbao-entity-operator-system \
  --create-namespace
```

### From a checkout

The Helm chart is the supported configurable installation surface:

```sh
helm upgrade --install openbao-entity-operator \
  charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace
```

The chart packages the CRDs and generated controller permissions. See the
[chart README](charts/openbao-entity-operator/README.md) for image, metrics,
and Prometheus ServiceMonitor values.

The generated Kustomize bundle remains available when a standalone manifest is
preferred:

```sh
make build-installer
kubectl apply -f dist/install.yaml
```

For local development, `make run` uses the active kubeconfig context. Build an image with `make docker-build IMG=...`, publish it with `make docker-push IMG=...`, package the chart with `make helm-package`, and render an install bundle for that image with `make build-installer IMG=...`. `make deploy` installs or upgrades the Helm release in the active context.

## Development

```sh
make check
make kind-e2e
make kind-down
```

`make check` regenerates CRDs, deepcopy code, chart assets, and the CRD API reference, checks formatting, runs vet and unit tests, runs lint and Helm chart checks, validates the checked-in OpenBao OpenAPI reference, renders the installation manifests, and builds the strict documentation site. `make kind-e2e` builds the operator, installs its Helm chart, and tests the live OpenBao lifecycle in an isolated Kind cluster. See [the documentation map](docs/index.md) for product scope, design stories, architecture, operations, research, and verification details. Repository operating rules live in [AGENTS.md](AGENTS.md).

## License

Apache License 2.0. See [LICENSE](LICENSE).
