# Resource guides

These pages explain how each custom resource behaves in practice, including
examples, ownership semantics, and OpenBao-specific caveats. Use the generated
[API reference](../reference/api.md) for exact field definitions, defaults,
enums, and validation rules.

## Connections

- [OpenBaoConnection](openbao-connection.md)

## Policies and identities

- [OpenBaoPolicy](openbao-policy.md)
- [OpenBaoKubernetesAuthRole](openbao-kubernetes-auth-role.md)
- [OpenBaoEntity](openbao-entity.md)
- [OpenBaoEntityAlias](openbao-entity-alias.md)
- [OpenBaoGroupAlias](openbao-identity-configuration.md#openbaogroupalias)
- [OpenBaoTokenRole](openbao-identity-configuration.md#openbaotokenrole)
- [OpenBaoAppRole](openbao-identity-configuration.md#openbaoapprole)
- [OpenBaoPasswordPolicy](openbao-identity-configuration.md#openbaopasswordpolicy)
- [OpenBaoPersona](openbao-identity-configuration.md#openbaopersona)
- [OpenBaoMFAMethod](openbao-identity-configuration.md#openbaomfamethod)
- [OpenBaoMFALoginEnforcement](openbao-identity-configuration.md#openbaomfaloginenforcement)

## Groups and relationships

- [OpenBaoGroup](openbao-group.md)
- [OpenBaoGroupMembership](openbao-group-membership.md)

## OpenBao configuration

- [Mounts, namespaces, audit, quotas, workflows, and plugins](openbao-configuration.md)
- [OIDC configuration resources](openbao-oidc.md)
- [System configuration resources](openbao-configuration.md#system-configuration)

The [resource index](../reference/resources.md) links each guide to its
generated schema section.
