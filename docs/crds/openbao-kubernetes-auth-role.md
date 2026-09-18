# OpenBaoKubernetesAuthRole

`OpenBaoKubernetesAuthRole` manages one role in an already enabled and
configured OpenBao Kubernetes Auth mount. The Kubernetes `metadata.name` is
used as the OpenBao role name. The resource does not enable the auth method or
configure its Kubernetes TokenReview credentials; those remain platform
administration tasks.

## Example

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoKubernetesAuthRole
metadata:
  name: payments-workload
  namespace: payments
spec:
  connectionRef:
    name: openbao
  mountPath: kubernetes
  boundServiceAccountNames:
    - payments
  boundServiceAccountNamespaces:
    - payments
  tokenPolicies:
    - payments-read
  tokenTTL: 1h
  tokenMaxTTL: 4h
```

The connection must be ready and its OpenBao credential must be authorized to
read and write the selected `auth/<mount>/role/<name>` path. Keep that
connection platform-owned when the role is part of a tenant boundary; the
role's Kubernetes RBAC permission does not grant access to the connection or
its credential.

## Role ownership

Roles default to `creationPolicy: Create` and `deletionPolicy: Orphan`:

- `Create` creates a missing role and rejects an existing unacquired role;
- `Adopt` manages an existing role and rejects a missing role; and
- `CreateOrAdopt` creates when missing and adopts when present.

Set `deletionPolicy: Delete` only when deleting the Kubernetes resource should
also delete the OpenBao role. A Delete-policy resource retains its finalizer
when its connection or credential is unavailable, so external cleanup can
resume after the dependency is restored.

`mountPath` and `connectionRef` are immutable. A role can be moved to another
auth mount or connection by deleting and recreating the Kubernetes resource.
The mount path is relative to `auth/`; for example, `custom-kubernetes` maps
to `auth/custom-kubernetes/role/<metadata.name>`.

## Bound identities and token lifetime

`boundServiceAccountNames` and `boundServiceAccountNamespaces` are required
non-empty sets. OpenBao's `*` wildcard is supported by the API, but it should
only be used when the connection's OpenBao policy deliberately permits that
boundary. `tokenPolicies` is also a set and the controller compares it
deterministically.

Durations are sent to OpenBao as whole seconds through the `token_ttl`,
`token_max_ttl`, and `token_period` fields. An omitted or zero value leaves
the corresponding OpenBao default or disabled behavior in place. The status
contains only the mount, role name, normalized configuration hash, and
conditions; it never contains a token or ServiceAccount JWT.

## Drift and tenancy

Periodic checks use `driftDetectionInterval`. If an acquired role is changed
outside Kubernetes, the operator restores the declared bindings, policies,
and token lifetime.

The OpenBao ACL on the connection remains the external authorization boundary.
For a tenant installation, combine same-namespace resources and the published
tenant-author RBAC profile with a platform-owned connection and a least-
privilege OpenBao policy. See the [multi-tenancy guide](../reference/multi-tenancy.md)
for the supported model.

See the [generated OpenBaoKubernetesAuthRole schema](../reference/api.md#openbaokubernetesauthrole)
for exact validation and defaults.
