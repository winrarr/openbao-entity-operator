# API Reference

## Packages
- [openbao.openbao-operator.io/v1alpha1](#openbaoopenbao-operatoriov1alpha1)


## openbao.openbao-operator.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the openbao v1alpha1 API group.

### Resource Types
- [OpenBaoConnection](#openbaoconnection)
- [OpenBaoEntity](#openbaoentity)
- [OpenBaoEntityAlias](#openbaoentityalias)
- [OpenBaoGroup](#openbaogroup)
- [OpenBaoGroupMembership](#openbaogroupmembership)
- [OpenBaoKubernetesAuthRole](#openbaokubernetesauthrole)
- [OpenBaoPolicy](#openbaopolicy)



#### AppRoleAuthSpec



AppRoleAuthSpec defines how an OpenBaoConnection uses the AppRole auth
method. Both credential references are reread when a new login is needed.



_Appears in:_
- [OpenBaoConnectionSpec](#openbaoconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mountPath` _string_ | MountPath is the auth mount path without the leading auth/ prefix. | approle | Pattern: `^[^/[:space:]]+([/][^/[:space:]]+)*$` <br />Optional: \{\} <br /> |
| `roleIDSecretRef` _[SecretKeyReference](#secretkeyreference)_ | RoleIDSecretRef references a same-namespace Secret containing the AppRole<br />role ID. The key defaults to role-id. |  |  |
| `secretIDSecretRef` _[SecretKeyReference](#secretkeyreference)_ | SecretIDSecretRef references a same-namespace Secret containing the AppRole<br />Secret ID. The key defaults to secret-id. |  |  |


#### CreationPolicy

_Underlying type:_ _string_

CreationPolicy controls how an external entity is acquired.



_Appears in:_
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupSpec](#openbaogroupspec)
- [OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)
- [OpenBaoPolicySpec](#openbaopolicyspec)

| Field | Description |
| --- | --- |
| `Create` |  |
| `Adopt` |  |
| `CreateOrAdopt` |  |


#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy controls what happens to an external entity on deletion.



_Appears in:_
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupSpec](#openbaogroupspec)
- [OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)
- [OpenBaoPolicySpec](#openbaopolicyspec)

| Field | Description |
| --- | --- |
| `Delete` |  |
| `Orphan` |  |


#### KubernetesAuthSpec



KubernetesAuthSpec defines how an OpenBaoConnection uses the Kubernetes auth
method. The operator reads its projected ServiceAccount JWT from the standard
in-cluster token path and never stores that JWT in Kubernetes status.



_Appears in:_
- [OpenBaoConnectionSpec](#openbaoconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mountPath` _string_ | MountPath is the OpenBao auth mount path without the leading auth/ prefix. | kubernetes | Pattern: `^[^/[:space:]]+([/][^/[:space:]]+)*$` <br />Optional: \{\} <br /> |
| `role` _string_ | Role is the role configured in the OpenBao Kubernetes auth method. |  | MaxLength: 256 <br />MinLength: 1 <br />Pattern: `^[^[:space:]]+$` <br /> |


#### OpenBaoConnection



OpenBaoConnection is the Schema for the openbaoconnections API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoConnection` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[OpenBaoConnectionSpec](#openbaoconnectionspec)_ | spec defines the desired state of OpenBaoConnection |  | Required: \{\} <br /> |


#### OpenBaoConnectionReference



OpenBaoConnectionReference identifies a same-namespace OpenBaoConnection.



_Appears in:_
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupSpec](#openbaogroupspec)
- [OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)
- [OpenBaoPolicySpec](#openbaopolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the OpenBaoConnection resource name. |  | MinLength: 1 <br /> |


#### OpenBaoConnectionSpec



OpenBaoConnectionSpec defines the desired state of OpenBaoConnection.



_Appears in:_
- [OpenBaoConnection](#openbaoconnection)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `address` _string_ | Address is the OpenBao API address without the /v1 API prefix. |  | MinLength: 1 <br />Pattern: `^https?://` <br /> |
| `namespace` _string_ | Namespace is an optional absolute or relative OpenBao namespace path.<br />An empty value targets the root namespace. The value is sent as the<br />X-Vault-Namespace request header. |  | Pattern: `^$\|^[^/[:space:]]+([/][^/[:space:]]+)*$` <br />Optional: \{\} <br /> |
| `tokenSecretRef` _[SecretKeyReference](#secretkeyreference)_ | TokenSecretRef references a same-namespace Secret containing an OpenBao token.<br />Exactly one of TokenSecretRef, KubernetesAuth, and AppRole must be configured. |  | Optional: \{\} <br /> |
| `kubernetesAuth` _[KubernetesAuthSpec](#kubernetesauthspec)_ | KubernetesAuth logs the operator into OpenBao with its projected Kubernetes<br />ServiceAccount token. The Kubernetes auth method must already be enabled<br />and configured at the selected mount path.<br />Exactly one of TokenSecretRef, KubernetesAuth, and AppRole must be configured. |  | Optional: \{\} <br /> |
| `appRole` _[AppRoleAuthSpec](#approleauthspec)_ | AppRole logs the operator into OpenBao with an AppRole role ID and Secret ID<br />read from same-namespace Secrets. The AppRole auth method and role must<br />already be configured in OpenBao.<br />Exactly one of TokenSecretRef, KubernetesAuth, and AppRole must be configured. |  | Optional: \{\} <br /> |
| `caBundleSecretRef` _[SecretKeyReference](#secretkeyreference)_ | CABundleSecretRef optionally references a same-namespace Secret containing a PEM CA bundle.<br />The key defaults to ca.crt when omitted. |  | Optional: \{\} <br /> |
| `requestTimeout` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | RequestTimeout bounds each request made to OpenBao. | 30s | Optional: \{\} <br /> |


#### OpenBaoEntity



OpenBaoEntity is the Schema for the openbaoentities API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoEntity` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[OpenBaoEntitySpec](#openbaoentityspec)_ | spec defines the desired state of OpenBaoEntity |  | Required: \{\} <br /> |


#### OpenBaoEntityAlias



OpenBaoEntityAlias is the Schema for the openbaoentityaliases API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoEntityAlias` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[OpenBaoEntityAliasSpec](#openbaoentityaliasspec)_ | spec defines the desired state of OpenBaoEntityAlias |  | Required: \{\} <br /> |


#### OpenBaoEntityAliasSpec



OpenBaoEntityAliasSpec defines the desired state of an OpenBao entity alias.



_Appears in:_
- [OpenBaoEntityAlias](#openbaoentityalias)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `entityRef` _[OpenBaoEntityReference](#openbaoentityreference)_ | EntityRef selects the OpenBaoEntity receiving this alias in the same namespace. |  |  |
| `mountAccessor` _string_ | MountAccessor is the OpenBao auth-method mount accessor for this alias. |  | MinLength: 1 <br /> |
| `name` _string_ | Name is the name presented by the auth method for this alias. |  | MinLength: 1 <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external alias is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external alias is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |


#### OpenBaoEntityReference



OpenBaoEntityReference identifies a same-namespace OpenBaoEntity.



_Appears in:_
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoGroupMembershipSpec](#openbaogroupmembershipspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the OpenBaoEntity resource name. |  | MinLength: 1 <br /> |


#### OpenBaoEntitySpec



OpenBaoEntitySpec defines the desired state of OpenBaoEntity.



_Appears in:_
- [OpenBaoEntity](#openbaoentity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external entity is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external entity is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `policies` _string array_ | Policies is the desired set of OpenBao ACL policy names. |  | Optional: \{\} <br /> |
| `disabled` _boolean_ | Disabled prevents tokens associated with the entity from being used. |  | Optional: \{\} <br /> |


#### OpenBaoGroup



OpenBaoGroup is the Schema for the openbaogroups API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoGroup` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoGroupSpec](#openbaogroupspec)_ |  |  |  |


#### OpenBaoGroupMembership



OpenBaoGroupMembership is the Schema for the openbaogroupmemberships API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoGroupMembership` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoGroupMembershipSpec](#openbaogroupmembershipspec)_ |  |  |  |


#### OpenBaoGroupMembershipSpec



OpenBaoGroupMembershipSpec defines one explicit membership edge.



_Appears in:_
- [OpenBaoGroupMembership](#openbaogroupmembership)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `groupRef` _[OpenBaoGroupReference](#openbaogroupreference)_ | GroupRef selects the parent OpenBaoGroup in the same namespace. |  |  |
| `entityRef` _[OpenBaoEntityReference](#openbaoentityreference)_ | EntityRef selects an OpenBaoEntity to add to the parent group. |  | Optional: \{\} <br /> |
| `memberGroupRef` _[OpenBaoGroupReference](#openbaogroupreference)_ | MemberGroupRef selects an OpenBaoGroup to add as a subgroup of the parent group. |  | Optional: \{\} <br /> |


#### OpenBaoGroupReference



OpenBaoGroupReference identifies a same-namespace OpenBaoGroup.



_Appears in:_
- [OpenBaoGroupMembershipSpec](#openbaogroupmembershipspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the OpenBaoGroup resource name. |  | MinLength: 1 <br /> |


#### OpenBaoGroupSpec



OpenBaoGroupSpec defines the desired state of an OpenBao identity group.



_Appears in:_
- [OpenBaoGroup](#openbaogroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `type` _[OpenBaoGroupType](#openbaogrouptype)_ | Type selects an internal or external OpenBao group. Membership resources<br />are supported for internal groups. | Internal | Enum: [Internal External] <br />Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external group is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external group is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `policies` _string array_ | Policies is the desired set of OpenBao ACL policy names. |  | Optional: \{\} <br /> |


#### OpenBaoGroupType

_Underlying type:_ _string_

OpenBaoGroupType identifies the OpenBao identity group membership model.



_Appears in:_
- [OpenBaoGroupSpec](#openbaogroupspec)

| Field | Description |
| --- | --- |
| `Internal` |  |
| `External` |  |


#### OpenBaoKubernetesAuthRole



OpenBaoKubernetesAuthRole is the Schema for the openbaokubernetesauthroles API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoKubernetesAuthRole` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)_ | spec defines the desired state of OpenBaoKubernetesAuthRole |  | Required: \{\} <br /> |


#### OpenBaoKubernetesAuthRoleSpec



OpenBaoKubernetesAuthRoleSpec defines the desired state of a role in an
OpenBao Kubernetes Auth mount.



_Appears in:_
- [OpenBaoKubernetesAuthRole](#openbaokubernetesauthrole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `mountPath` _string_ | MountPath is the OpenBao Kubernetes Auth mount path without the leading<br />auth/ prefix. The mount must already be enabled and configured. | kubernetes | Pattern: `^[^/[:space:]]+([/][^/[:space:]]+)*$` <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external role is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external role is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |
| `boundServiceAccountNames` _string array_ | BoundServiceAccountNames lists the Kubernetes ServiceAccounts allowed to use the role.<br />OpenBao also accepts * as a wildcard; use it only when the connection's OpenBao<br />policy deliberately permits that boundary. |  | MinItems: 1 <br /> |
| `boundServiceAccountNamespaces` _string array_ | BoundServiceAccountNamespaces lists the Kubernetes namespaces allowed to use the role.<br />OpenBao also accepts * as a wildcard; use it only when the connection's OpenBao<br />policy deliberately permits that boundary. |  | MinItems: 1 <br /> |
| `tokenPolicies` _string array_ | TokenPolicies is the set of OpenBao ACL policies attached to tokens issued by the role. |  | Optional: \{\} <br /> |
| `tokenTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenTTL is the initial token lifetime. A zero or omitted value uses OpenBao's default. |  | Optional: \{\} <br /> |
| `tokenMaxTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenMaxTTL is the maximum token lifetime. A zero or omitted value uses OpenBao's default. |  | Optional: \{\} <br /> |
| `tokenPeriod` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenPeriod gives issued tokens a fixed renewable period. A zero or omitted value disables it. |  | Optional: \{\} <br /> |


#### OpenBaoPolicy



OpenBaoPolicy is the Schema for the openbaopolicies API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoPolicy` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[OpenBaoPolicySpec](#openbaopolicyspec)_ | spec defines the desired state of OpenBaoPolicy |  | Required: \{\} <br /> |


#### OpenBaoPolicySpec



OpenBaoPolicySpec defines the desired state of an OpenBao ACL policy.



_Appears in:_
- [OpenBaoPolicy](#openbaopolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external policy is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external policy is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |
| `rules` _string_ | Rules is the raw HCL or JSON OpenBao ACL policy document. |  | MinLength: 1 <br /> |


#### SecretKeyReference



SecretKeyReference identifies a key in a same-namespace Secret.



_Appears in:_
- [AppRoleAuthSpec](#approleauthspec)
- [OpenBaoConnectionSpec](#openbaoconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret resource name. |  | MinLength: 1 <br /> |
| `key` _string_ | Key is the Secret data key. Defaults depend on the referencing field:<br />token for token references, role-id for AppRole role IDs, secret-id for<br />AppRole Secret IDs, and ca.crt for CA bundle references. |  | Optional: \{\} <br /> |
