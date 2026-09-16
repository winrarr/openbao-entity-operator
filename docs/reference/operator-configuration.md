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

The chart does not configure OpenBao auth methods or roles. Those must be
prepared by an OpenBao administrator.

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
