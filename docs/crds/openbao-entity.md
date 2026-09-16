# OpenBaoEntity

`OpenBaoEntity` manages one OpenBao identity entity. Its Kubernetes
`metadata.name` is the desired OpenBao entity name.

## Example

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  metadata:
    owner: platform
    team: payments
  policies:
    - default
    - payments
  disabled: false
```

The controller acquires an entity by its stable OpenBao ID, updates metadata,
policy names, and disabled state, and records the ID in status. Once acquired,
the ID prevents a later rename or replacement with another entity from being
treated as the same object.

## Existing entities and deletion

Use `creationPolicy: Adopt` or `CreateOrAdopt` to manage an existing entity.
The default `deletionPolicy: Orphan` leaves the OpenBao entity in place. Set
`deletionPolicy: Delete` only when removing the Kubernetes object should also
remove the external entity and its aliases.

External deletion is detected during reconciliation. The controller recreates
or reacquires the entity according to the creation policy and updates status
with the new observed ID when appropriate.

See the [generated OpenBaoEntity schema](../reference/api.md#openbaoentity) for
the exact fields and [entity ownership](../reference/deletion-and-ownership.md)
for lifecycle details.
