# Status and conditions

Every managed resource reports reconciliation state through Kubernetes status.
The `Ready` condition is the quickest way to determine whether the desired
OpenBao state is available.

- `Ready=True` means the latest desired state was applied successfully.
- `Ready=False` means reconciliation needs attention; inspect `reason` and
  `message`.
- `CleanupRequired=True` means a Delete-policy resource is waiting for the
  connection, credentials, or OpenBao API access needed to remove its external
  object. The resource intentionally remains `Terminating` until cleanup can
  be proven.
- `Stalled=True` means reconciliation is blocked. During deletion, inspect
  `CleanupRequired` and restore the named dependency before considering any
  manual finalizer action.
- `observedGeneration` identifies the resource generation reflected in status.
  If it lags behind `metadata.generation`, reconciliation has not caught up.

## Resource-specific status

- `OpenBaoConnection` reports authentication and OpenBao health details.
- `OpenBaoPolicy` reports the acquired name, observed policy version, and
  SHA-256 hash of the returned rules.
- `OpenBaoEntity` reports the stable OpenBao entity ID.
- `OpenBaoEntityAlias` reports the alias ID and canonical entity ID.
- `OpenBaoGroup` reports the stable OpenBao group ID.
- `OpenBaoGroupMembership` reports the parent group ID, member ID, and member
  type.

Inspect status with:

```sh
kubectl get openbaopolicy,openbaoentity,openbaoentityalias,openbaogroup \
  -n platform
kubectl get openbaopolicy/payments -n platform -o yaml
kubectl describe openbaoentity/payments -n platform
```

During deletion, a resource may remain `Terminating` while its finalizer waits
for external cleanup. See [Deletion and ownership](deletion-and-ownership.md)
and [Troubleshooting](troubleshooting.md) for recovery steps.
