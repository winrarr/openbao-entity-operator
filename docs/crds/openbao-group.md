# OpenBaoGroup

`OpenBaoGroup` manages an OpenBao identity group. Its Kubernetes
`metadata.name` is the desired OpenBao group name.

## Example

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroup
metadata:
  name: platform
spec:
  connectionRef:
    name: openbao
  type: internal
  policies:
    - default
    - platform
```

Internal groups can be targeted by `OpenBaoGroupMembership` resources. External
groups can be reconciled as group objects, but membership claims are intended
for internal groups because OpenBao owns external group membership differently.

## Group membership claims

```yaml
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

The group controller preserves remote members that are not claimed by a
membership resource. It removes only relationships represented by deleted or
changed membership claims.

Groups support the same explicit creation and deletion policies as entities.
Use `creationPolicy: Adopt` for a pre-existing group and leave the default
`deletionPolicy: Orphan` unless external deletion is intentional.

See the [generated OpenBaoGroup schema](../reference/api.md#openbaogroup) and
[group membership guide](openbao-group-membership.md).
