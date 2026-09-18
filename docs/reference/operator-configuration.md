# Operator configuration

The Helm chart is the supported configurable installation surface. Render the
chart before applying changes:

```sh
helm template openbao-entity-operator charts/openbao-entity-operator \
  --namespace openbao-entity-operator-system \
  --include-crds
```

## Image and replicas

Set `image.repository`, `image.tag`, or `image.digest` to choose the controller
image. A digest takes precedence over the tag. `replicaCount` may be increased
for availability; leader election ensures only one replica performs active
reconciliation at a time.

## ServiceAccount and authentication

The chart enables `serviceAccount.automountServiceAccountToken` by default.
Keep it enabled when any connection uses Kubernetes Auth. It may be disabled
when all connections use static token Secrets or AppRole.

The chart does not configure the authentication prerequisites used to connect
to OpenBao, such as TokenReview credentials or bootstrap credentials. After the
operator is installed and a connection is Ready, `OpenBaoAuthMethod` and the
role resources can declaratively configure durable auth state when the
connection's OpenBao ACL allows it.

## Kubernetes watch scope

`watchNamespaces` is an optional list of Kubernetes namespaces. When it is
empty, the manager watches all namespaces for compatibility with the default
trusted-platform installation. When it contains one or more namespaces, the
manager cache watches only those namespaces and the chart binds the manager
permissions with a `RoleBinding` in each listed namespace instead of using the
manager `ClusterRoleBinding`.

```yaml
watchNamespaces:
  - team-a
  - team-b
```

This setting is intended for separate operator installations or other
platform-controlled scopes. It does not decide which users may submit
`OpenBaoConnection` resources, which resource kinds they may use, or which
OpenBao namespace a credential can access. Use Kubernetes RBAC or admission
policy for those authoring rules, and use an OpenBao namespace with a
least-privilege policy for the external API boundary.

The chart also publishes an unbound `ClusterRole` named
`<release-name>-tenant-author-role`. Bind it with a namespace `RoleBinding` to
each tenant ServiceAccount that should author resources. It grants CRUD access
to the supported entity and durable configuration resources; it omits
connections, connection status/finalizers, and Secrets. Keep the platform-owned
connection and any static credential Secret outside the tenant's authoring
permissions. Kubernetes Auth is preferred because it avoids a static
connection credential Secret altogether.

Example:

```sh
kubectl -n team-a create rolebinding team-a-openbao-author \
  --clusterrole=openbao-entity-operator-tenant-author-role \
  --serviceaccount=team-a:team-a-operator
```

This role is a permission profile, not an admission policy. Use platform RBAC
or admission policy to control tenant naming, deletion policies, and which
ServiceAccounts receive it.

## Metrics

Secure metrics are enabled by default. The chart creates the controller-runtime
metrics service, authentication RBAC, and an optional reader ClusterRole. A
Prometheus ServiceMonitor is disabled by default because the chart cannot infer
the Prometheus service account safely.

Enable it only with the matching service account and labels for the Prometheus
installation:

```yaml
metrics:
  serviceMonitor:
    enabled: true
    namespace: monitoring
    labels:
      release: prometheus
```

The generated metrics certificate is intended for development. Configure an
existing TLS Secret and the appropriate authorization path for production
observability.

## Security defaults

The chart runs the manager as non-root, drops Linux capabilities, disallows
privilege escalation, and uses a read-only root filesystem. Preserve these
defaults unless the deployment environment requires a documented exception.
