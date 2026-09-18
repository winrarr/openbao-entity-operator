# OpenBao configuration resources

These resources configure an OpenBao server that already exists. They do not
deploy OpenBao or own its storage, HA, initialization, unseal, or upgrade
lifecycle. Use the generated [API reference](../reference/api.md) for exact
field types and validation.

All resources use an immutable, same-namespace `connectionRef`, explicit
creation/adoption policy, periodic drift detection, and `Orphan` deletion by
default. The OpenBao token behind the connection must have permission for the
specific endpoint.

## OpenBaoAuthMethod

`OpenBaoAuthMethod` enables an auth method at `spec.path` and applies durable
mount tuning. For example:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoAuthMethod
metadata:
  name: kubernetes
spec:
  connectionRef:
    name: openbao
  path: kubernetes
  type: kubernetes
  defaultLeaseTTL: 1h
  maxLeaseTTL: 24h
```

The Kubernetes Auth TokenReview configuration and any auth credentials remain
OpenBao-side platform configuration. This resource manages the mount, not the
OpenBao server or Kubernetes API integration behind it.

## OpenBaoSecretEngine

`OpenBaoSecretEngine` enables and tunes a secret engine. It manages the engine
configuration, not arbitrary secret data or generated credentials:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoSecretEngine
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  path: payments
  type: kv
  options:
    version: "2"
```

Use `deletionPolicy: Delete` only when deleting the Kubernetes resource should
disable the external engine. `Orphan` is the safer default.

## OpenBaoNamespace

`OpenBaoNamespace` manages metadata for an existing child OpenBao namespace.
It does not deploy the namespace's server or change the connection namespace
used by the Kubernetes resource:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoNamespace
metadata:
  name: platform-production
spec:
  connectionRef:
    name: root-openbao
  path: platform/production
  customMetadata:
    owner: platform
```

The connection used to manage the namespace must already be authorized to
access the parent namespace.

## OpenBaoAuditDevice

`OpenBaoAuditDevice` registers an audit backend and its options. File paths,
socket endpoints, and the runtime environment are external to the operator:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoAuditDevice
metadata:
  name: file
spec:
  connectionRef:
    name: openbao
  path: file
  type: file
  options:
    file_path: /var/log/openbao/audit.log
```

Audit devices can have high operational impact. Prefer `Orphan` until the
external deletion behavior has been explicitly reviewed.

## OpenBaoRateLimitQuota

`OpenBaoRateLimitQuota` manages one named rate-limit quota. `spec.rate` is a
positive decimal encoded as a string so Kubernetes serialization does not
silently round it:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoRateLimitQuota
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  type: path
  path: payments
  rate: "10"
  interval: 1s
```

## OpenBaoWorkflow

`OpenBaoWorkflow` stores a durable workflow definition at `spec.path`. The
workflow text is declarative configuration; executing it is not part of
reconciliation:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoWorkflow
metadata:
  name: payments-rotation
spec:
  connectionRef:
    name: openbao
  path: payments/rotation
  workflow: |
    # OpenBao workflow definition
```

Use `cas` and `casRequired` when a workflow should be protected against
concurrent external writes.

## OpenBaoPlugin

`OpenBaoPlugin` manages a catalog registration for a plugin artifact that is
already available to OpenBao. It does not install, download, verify, or run
the plugin binary:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPlugin
metadata:
  name: payments-plugin
spec:
  connectionRef:
    name: openbao
  name: payments-plugin
  type: secret
  command: openbao-plugin-payments
  sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

The plugin catalog registration is deleted only with `deletionPolicy: Delete`.

## System configuration

The following resources manage durable OpenBao system settings. Each uses the
same connection, ownership, drift, status, and default-orphan semantics as the
other configuration resources.

- `OpenBaoCORSConfiguration` manages the singleton CORS configuration.
- `OpenBaoAuditRequestHeader` manages one audit request-header HMAC setting.
- `OpenBaoUIHeader` manages one UI response header and its values.
- `OpenBaoRateLimitQuotaConfiguration` manages global quota logging, response
  headers, and exempt paths. OpenBao has no delete operation for this
  singleton, so use `deletionPolicy: Orphan`.
- `OpenBaoLogger` manages the global logger when `spec.name` is empty or a
  named subsystem logger otherwise.
- `OpenBaoEncryptionKeyConfiguration` and
  `OpenBaoKeyringRotationConfiguration` manage automatic rotation settings.
  They do not perform an immediate key rotation and have no external delete
  operation; use `deletionPolicy: Orphan`.

For example:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoLogger
metadata:
  name: audit-logger
spec:
  connectionRef:
    name: openbao
  name: audit
  level: info
```

OpenBao supports one effective singleton configuration per connection for the
CORS, global quota, and rotation resources. The operator does not enforce
singleton uniqueness across Kubernetes objects; platform policy should ensure
that one resource owns each singleton.

Named logger resources target logger subsystems that must already exist in
OpenBao. Use `creationPolicy: Adopt` (or `CreateOrAdopt`) when managing an
existing subsystem logger; an empty logger name targets OpenBao's global logger
configuration. Resources backed by endpoints without an OpenBao delete
operation must use `deletionPolicy: Orphan`.
