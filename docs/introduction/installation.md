# Installation

The recommended production installation is the Helm chart published to the
GitHub Container Registry.

## Prerequisites

- an existing OpenBao instance reachable from the operator Pod;
- permission to install CRDs, RBAC, and the operator deployment;
- an OpenBao token policy or Kubernetes Auth role that grants the operations
  required by the resources you will manage;
- a same-namespace Kubernetes Secret when using token or AppRole authentication,
  including the AppRole role ID and Secret ID, or a configured OpenBao
  Kubernetes Auth role when using Kubernetes Auth.

The operator does not install OpenBao, enable auth methods, configure
TokenReview credentials, or create OpenBao roles and policies for its own
connection. Configure those prerequisites separately.

## Install from OCI

```sh
helm upgrade --install openbao-entity-operator \
  oci://ghcr.io/winrarr/charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace \
  --version <chart-version>
```

The chart packages the CRDs and generated controller permissions. Pin a
published chart version in production.

## Install from a checkout

```sh
helm upgrade --install openbao-entity-operator \
  charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace
```

The generated Kustomize bundle remains available for workflows that prefer a
standalone manifest:

```sh
make build-installer
kubectl apply -f dist/install.yaml
```

## Useful chart settings

The chart exposes settings for:

- the operator image repository, tag, or digest;
- resource requests and limits, scheduling, and security contexts;
- secure controller-runtime metrics and an optional ServiceMonitor;
- an existing ServiceAccount and projected token automounting;
- leader election and the health probe address;
- `watchNamespaces` to scope a Helm installation to explicit Kubernetes
  namespaces. In scoped mode the chart uses namespace RoleBindings for manager
  permissions; those namespaces must exist before the chart is installed.
- the unbound `<release-name>-tenant-author-role` ClusterRole for a
  platform-managed tenant authoring profile.

Inspect the [chart README](https://github.com/winrarr/openbao-entity-operator/blob/main/charts/openbao-entity-operator/README.md)
and `charts/openbao-entity-operator/values.yaml` in a checkout for the
complete values surface.
Keep ServiceAccount token automounting enabled when any connection uses
Kubernetes Auth.

For a tenant-scoped installation, combine `watchNamespaces` with an
OpenBao-native boundary:

```sh
helm upgrade --install openbao-entity-operator \
  charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --create-namespace \
  --set 'watchNamespaces[0]=team-a'
```

Use an OpenBao token, AppRole, or Kubernetes Auth role whose ACL policy is
limited to the corresponding OpenBao namespace. The operator does not create
or manage OpenBao namespaces.

Create the `OpenBaoConnection` and its credential material as platform-owned
objects. Bind `<release-name>-tenant-author-role` to tenant ServiceAccounts so
they can manage identity and policy resources without being able to read or
change connections or Secrets. See [Multi-tenancy](../reference/multi-tenancy.md)
for the complete boundary and verification model.

## Verify the installation

```sh
kubectl -n openbao-entity-operator-system get deployment,pods
kubectl -n openbao-entity-operator-system logs deployment/openbao-entity-operator
```

Then create an `OpenBaoConnection` and wait for `Ready=True` before creating
dependent resources.
