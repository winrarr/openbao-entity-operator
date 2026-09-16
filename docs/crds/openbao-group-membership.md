# OpenBaoGroupMembership

`OpenBaoGroupMembership` claims one relationship between an internal
`OpenBaoGroup` and either an `OpenBaoEntity` or another internal group.

## Entity membership

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

## Group nesting

Use `groupRef` together with `memberGroupRef` when the parent should contain an
internal subgroup:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupMembership
metadata:
  name: platform-security
spec:
  groupRef:
    name: platform
  memberGroupRef:
    name: security
```

Exactly one of `entityRef` and `memberGroupRef` must be set. References are
same-namespace and immutable. The parent and target resources must be ready
before the claim can be applied.

Membership resources do not create, adopt, or delete groups and entities. They
manage only their claimed edge. Deleting a claim removes that edge while
preserving unrelated remote members and the parent and target objects.

See the [generated OpenBaoGroupMembership schema](../reference/api.md#openbaogroupmembership)
for the exact validation rules.
