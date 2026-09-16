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



#### CreationPolicy

_Underlying type:_ _string_

CreationPolicy controls how an external entity is acquired.



_Appears in:_
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupSpec](#openbaogroupspec)

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

| Field | Description |
| --- | --- |
| `Delete` |  |
| `Orphan` |  |


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
| `tokenSecretRef` _[SecretKeyReference](#secretkeyreference)_ | TokenSecretRef references a same-namespace Secret containing an OpenBao token. |  |  |
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


#### SecretKeyReference



SecretKeyReference identifies a key in a same-namespace Secret.



_Appears in:_
- [OpenBaoConnectionSpec](#openbaoconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret resource name. |  | MinLength: 1 <br /> |
| `key` _string_ | Key is the Secret data key. It defaults to token for token references. |  | Optional: \{\} <br /> |
