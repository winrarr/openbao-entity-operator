# OpenBaoEntityAlias

`OpenBaoEntityAlias` binds an alias presented by an OpenBao auth method to an
`OpenBaoEntity` in the same Kubernetes namespace.

## Example

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntityAlias
metadata:
  name: payments-user
spec:
  connectionRef:
    name: openbao
  entityRef:
    name: payments
  mountAccessor: auth_kubernetes_12345678
  name: payments@example.com
```

`mountAccessor` is the OpenBao auth-method mount accessor, not the mount path.
`name` is the exact alias value supplied by that auth method. The referenced
entity must be `Ready=True` before the alias can be reconciled.

## Ownership behavior

The controller records the alias ID and canonical entity ID in status. It
corrects external canonical-entity drift and follows the referenced entity's
current ID. Existing aliases require `creationPolicy: Adopt` or
`CreateOrAdopt`; aliases default to `deletionPolicy: Orphan`.

Set `deletionPolicy: Delete` only when deleting the Kubernetes alias should
remove the external alias. Deleting an alias never deletes its entity.

See the [generated OpenBaoEntityAlias schema](../reference/api.md#openbaoentityalias)
for exact validation and the [ownership reference](../reference/deletion-and-ownership.md)
for dependency-loss behavior.
