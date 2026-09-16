# OpenBao Entity Operator Helm Chart

This chart installs the OpenBao Entity Operator controller, its generated CRDs,
controller RBAC, and optional Prometheus ServiceMonitor.

## Install from an OCI registry

```sh
helm install openbao-entity-operator \
  oci://ghcr.io/winrarr/charts/openbao-entity-operator \
  --version <chart-version>
```

## Install from this checkout

```sh
helm upgrade --install openbao-entity-operator \
  charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace
```

The chart defaults to the development image tag and secure controller-runtime
metrics, matching the local Kustomize installation. Set `image.repository`,
`image.tag`, or `image.digest` for a published image. A digest takes precedence
over the tag.

When secure metrics are enabled, the chart creates the authentication and
authorization RBAC needed by controller-runtime. It creates a metrics reader
ClusterRole but does not bind it by default because the chart cannot infer the
Prometheus service account. Configure `metrics.readerRole.binding` when using a
Prometheus ServiceMonitor.

CRDs are packaged under `crds/` and are synchronized from `config/crd/bases/`.
The manager ClusterRole is synchronized from `config/rbac/role.yaml`; chart
templates own names, labels, and service-account bindings around those generated
permissions.
