# API Reference

## Packages
- [openbao.openbao-operator.io/v1alpha1](#openbaoopenbao-operatoriov1alpha1)


## openbao.openbao-operator.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the openbao v1alpha1 API group.

### Resource Types
- [OpenBaoAppRole](#openbaoapprole)
- [OpenBaoAuditDevice](#openbaoauditdevice)
- [OpenBaoAuditRequestHeader](#openbaoauditrequestheader)
- [OpenBaoAuthMethod](#openbaoauthmethod)
- [OpenBaoCORSConfiguration](#openbaocorsconfiguration)
- [OpenBaoConnection](#openbaoconnection)
- [OpenBaoEncryptionKeyConfiguration](#openbaoencryptionkeyconfiguration)
- [OpenBaoEntity](#openbaoentity)
- [OpenBaoEntityAlias](#openbaoentityalias)
- [OpenBaoGroup](#openbaogroup)
- [OpenBaoGroupAlias](#openbaogroupalias)
- [OpenBaoGroupMembership](#openbaogroupmembership)
- [OpenBaoKeyringRotationConfiguration](#openbaokeyringrotationconfiguration)
- [OpenBaoKubernetesAuthRole](#openbaokubernetesauthrole)
- [OpenBaoLogger](#openbaologger)
- [OpenBaoMFALoginEnforcement](#openbaomfaloginenforcement)
- [OpenBaoMFAMethod](#openbaomfamethod)
- [OpenBaoNamespace](#openbaonamespace)
- [OpenBaoOIDCAssignment](#openbaooidcassignment)
- [OpenBaoOIDCClient](#openbaooidcclient)
- [OpenBaoOIDCConfig](#openbaooidcconfig)
- [OpenBaoOIDCKey](#openbaooidckey)
- [OpenBaoOIDCProvider](#openbaooidcprovider)
- [OpenBaoOIDCRole](#openbaooidcrole)
- [OpenBaoOIDCScope](#openbaooidcscope)
- [OpenBaoPasswordPolicy](#openbaopasswordpolicy)
- [OpenBaoPersona](#openbaopersona)
- [OpenBaoPlugin](#openbaoplugin)
- [OpenBaoPolicy](#openbaopolicy)
- [OpenBaoRateLimitQuota](#openbaoratelimitquota)
- [OpenBaoRateLimitQuotaConfiguration](#openbaoratelimitquotaconfiguration)
- [OpenBaoSecretEngine](#openbaosecretengine)
- [OpenBaoTokenRole](#openbaotokenrole)
- [OpenBaoUIHeader](#openbaouiheader)
- [OpenBaoWorkflow](#openbaoworkflow)



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
- [OpenBaoAppRoleSpec](#openbaoapprolespec)
- [OpenBaoAuditDeviceSpec](#openbaoauditdevicespec)
- [OpenBaoAuditRequestHeaderSpec](#openbaoauditrequestheaderspec)
- [OpenBaoAuthMethodSpec](#openbaoauthmethodspec)
- [OpenBaoCORSConfigurationSpec](#openbaocorsconfigurationspec)
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupAliasSpec](#openbaogroupaliasspec)
- [OpenBaoGroupSpec](#openbaogroupspec)
- [OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)
- [OpenBaoLoggerSpec](#openbaologgerspec)
- [OpenBaoMFALoginEnforcementSpec](#openbaomfaloginenforcementspec)
- [OpenBaoMFAMethodSpec](#openbaomfamethodspec)
- [OpenBaoMountSpec](#openbaomountspec)
- [OpenBaoNamespaceSpec](#openbaonamespacespec)
- [OpenBaoOIDCAssignmentSpec](#openbaooidcassignmentspec)
- [OpenBaoOIDCClientSpec](#openbaooidcclientspec)
- [OpenBaoOIDCKeySpec](#openbaooidckeyspec)
- [OpenBaoOIDCProviderSpec](#openbaooidcproviderspec)
- [OpenBaoOIDCRoleSpec](#openbaooidcrolespec)
- [OpenBaoOIDCScopeSpec](#openbaooidcscopespec)
- [OpenBaoPasswordPolicySpec](#openbaopasswordpolicyspec)
- [OpenBaoPersonaSpec](#openbaopersonaspec)
- [OpenBaoPluginSpec](#openbaopluginspec)
- [OpenBaoPolicySpec](#openbaopolicyspec)
- [OpenBaoRateLimitQuotaConfigurationSpec](#openbaoratelimitquotaconfigurationspec)
- [OpenBaoRateLimitQuotaSpec](#openbaoratelimitquotaspec)
- [OpenBaoRotationConfigurationSpec](#openbaorotationconfigurationspec)
- [OpenBaoSecretEngineSpec](#openbaosecretenginespec)
- [OpenBaoTokenRoleSpec](#openbaotokenrolespec)
- [OpenBaoUIHeaderSpec](#openbaouiheaderspec)
- [OpenBaoWorkflowSpec](#openbaoworkflowspec)

| Field | Description |
| --- | --- |
| `Create` |  |
| `Adopt` |  |
| `CreateOrAdopt` |  |


#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy controls what happens to an external entity on deletion.



_Appears in:_
- [OpenBaoAppRoleSpec](#openbaoapprolespec)
- [OpenBaoAuditDeviceSpec](#openbaoauditdevicespec)
- [OpenBaoAuditRequestHeaderSpec](#openbaoauditrequestheaderspec)
- [OpenBaoAuthMethodSpec](#openbaoauthmethodspec)
- [OpenBaoCORSConfigurationSpec](#openbaocorsconfigurationspec)
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupAliasSpec](#openbaogroupaliasspec)
- [OpenBaoGroupSpec](#openbaogroupspec)
- [OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)
- [OpenBaoLoggerSpec](#openbaologgerspec)
- [OpenBaoMFALoginEnforcementSpec](#openbaomfaloginenforcementspec)
- [OpenBaoMFAMethodSpec](#openbaomfamethodspec)
- [OpenBaoMountSpec](#openbaomountspec)
- [OpenBaoNamespaceSpec](#openbaonamespacespec)
- [OpenBaoOIDCAssignmentSpec](#openbaooidcassignmentspec)
- [OpenBaoOIDCClientSpec](#openbaooidcclientspec)
- [OpenBaoOIDCKeySpec](#openbaooidckeyspec)
- [OpenBaoOIDCProviderSpec](#openbaooidcproviderspec)
- [OpenBaoOIDCRoleSpec](#openbaooidcrolespec)
- [OpenBaoOIDCScopeSpec](#openbaooidcscopespec)
- [OpenBaoPasswordPolicySpec](#openbaopasswordpolicyspec)
- [OpenBaoPersonaSpec](#openbaopersonaspec)
- [OpenBaoPluginSpec](#openbaopluginspec)
- [OpenBaoPolicySpec](#openbaopolicyspec)
- [OpenBaoRateLimitQuotaConfigurationSpec](#openbaoratelimitquotaconfigurationspec)
- [OpenBaoRateLimitQuotaSpec](#openbaoratelimitquotaspec)
- [OpenBaoRotationConfigurationSpec](#openbaorotationconfigurationspec)
- [OpenBaoSecretEngineSpec](#openbaosecretenginespec)
- [OpenBaoTokenRoleSpec](#openbaotokenrolespec)
- [OpenBaoUIHeaderSpec](#openbaouiheaderspec)
- [OpenBaoWorkflowSpec](#openbaoworkflowspec)

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


#### OpenBaoAppRole



OpenBaoAppRole is the Schema for the openbaoapproles API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoAppRole` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoAppRoleSpec](#openbaoapprolespec)_ |  |  |  |


#### OpenBaoAppRoleSpec



OpenBaoAppRoleSpec defines a role in an already enabled OpenBao AppRole mount.
The resource manages role configuration only; Secret IDs are issued and
delivered by an external credential workflow.



_Appears in:_
- [OpenBaoAppRole](#openbaoapprole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `mountPath` _string_ | MountPath is the AppRole auth mount path without the leading auth/ prefix. | approle | Pattern: `^[^/[:space:]]+([/][^/[:space:]]+)*$` <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external role is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external role is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |
| `bindSecretID` _boolean_ | BindSecretID requires a Secret ID for login. |  | Optional: \{\} <br /> |
| `localSecretIDs` _boolean_ | LocalSecretIDs stores Secret IDs locally to this AppRole mount. |  | Optional: \{\} <br /> |
| `secretIDBoundCIDRs` _string array_ | SecretIDBoundCIDRs restricts Secret ID use to these client address ranges. |  | Optional: \{\} <br /> |
| `secretIDNumUses` _integer_ | SecretIDNumUses limits how many times a Secret ID may be used. Zero means unlimited. |  | Optional: \{\} <br /> |
| `secretIDTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | SecretIDTTL is the lifetime of issued Secret IDs. |  | Optional: \{\} <br /> |
| `tokenBoundCIDRs` _string array_ | TokenBoundCIDRs restricts issued tokens to these client address ranges. |  | Optional: \{\} <br /> |
| `tokenExplicitMaxTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenExplicitMaxTTL gives issued tokens an explicit maximum TTL. |  | Optional: \{\} <br /> |
| `tokenMaxTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenMaxTTL is the maximum lifetime of issued tokens. |  | Optional: \{\} <br /> |
| `tokenNoDefaultPolicy` _boolean_ | TokenNoDefaultPolicy prevents OpenBao from adding the default policy. |  | Optional: \{\} <br /> |
| `tokenNumUses` _integer_ | TokenNumUses limits how many times an issued token may be used. Zero means unlimited. |  | Optional: \{\} <br /> |
| `tokenPeriod` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenPeriod gives issued tokens a fixed renewable period. |  | Optional: \{\} <br /> |
| `tokenPolicies` _string array_ | TokenPolicies is the set of ACL policies attached to issued tokens. |  | Optional: \{\} <br /> |
| `tokenTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenTTL is the initial lifetime of issued tokens. |  | Optional: \{\} <br /> |
| `tokenType` _string_ | TokenType selects the type of token issued by the role. |  | Enum: [service batch] <br />Optional: \{\} <br /> |


#### OpenBaoAuditDevice









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoAuditDevice` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoAuditDeviceSpec](#openbaoauditdevicespec)_ |  |  |  |


#### OpenBaoAuditDeviceSpec



OpenBaoAuditDeviceSpec configures an OpenBao audit device.



_Appears in:_
- [OpenBaoAuditDevice](#openbaoauditdevice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `path` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `description` _string_ |  |  |  |
| `local` _boolean_ |  |  |  |
| `options` _object (keys:string, values:string)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoAuditRequestHeader









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoAuditRequestHeader` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoAuditRequestHeaderSpec](#openbaoauditrequestheaderspec)_ |  |  |  |


#### OpenBaoAuditRequestHeaderSpec



OpenBaoAuditRequestHeaderSpec configures one audit request header.



_Appears in:_
- [OpenBaoAuditRequestHeader](#openbaoauditrequestheader)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `header` _string_ |  |  |  |
| `hmac` _boolean_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoAuthMethod









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoAuthMethod` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoAuthMethodSpec](#openbaoauthmethodspec)_ |  |  |  |


#### OpenBaoAuthMethodSpec



OpenBaoAuthMethodSpec enables and tunes an OpenBao auth method.



_Appears in:_
- [OpenBaoAuthMethod](#openbaoauthmethod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `path` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `description` _string_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `local` _boolean_ |  |  |  |
| `sealWrap` _boolean_ |  |  |  |
| `externalEntropyAccess` _boolean_ |  |  |  |
| `pluginName` _string_ |  |  |  |
| `pluginVersion` _string_ |  |  |  |
| `options` _object (keys:string, values:string)_ |  |  |  |
| `config` _object (keys:string, values:string)_ |  |  |  |
| `defaultLeaseTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `maxLeaseTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `listingVisibility` _string_ |  |  |  |
| `tokenType` _string_ |  |  |  |
| `passthroughRequestHeaders` _string array_ |  |  |  |
| `allowedResponseHeaders` _string array_ |  |  |  |


#### OpenBaoCORSConfiguration









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoCORSConfiguration` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoCORSConfigurationSpec](#openbaocorsconfigurationspec)_ |  |  |  |


#### OpenBaoCORSConfigurationSpec



OpenBaoCORSConfigurationSpec configures the singleton OpenBao CORS policy.



_Appears in:_
- [OpenBaoCORSConfiguration](#openbaocorsconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `enable` _boolean_ |  |  |  |
| `allowCredentials` _boolean_ |  |  |  |
| `allowedHeaders` _string array_ |  |  |  |
| `allowedOrigins` _string array_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


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
- [OpenBaoAppRoleSpec](#openbaoapprolespec)
- [OpenBaoAuditDeviceSpec](#openbaoauditdevicespec)
- [OpenBaoAuditRequestHeaderSpec](#openbaoauditrequestheaderspec)
- [OpenBaoAuthMethodSpec](#openbaoauthmethodspec)
- [OpenBaoCORSConfigurationSpec](#openbaocorsconfigurationspec)
- [OpenBaoEntityAliasSpec](#openbaoentityaliasspec)
- [OpenBaoEntitySpec](#openbaoentityspec)
- [OpenBaoGroupAliasSpec](#openbaogroupaliasspec)
- [OpenBaoGroupSpec](#openbaogroupspec)
- [OpenBaoKubernetesAuthRoleSpec](#openbaokubernetesauthrolespec)
- [OpenBaoLoggerSpec](#openbaologgerspec)
- [OpenBaoMFALoginEnforcementSpec](#openbaomfaloginenforcementspec)
- [OpenBaoMFAMethodSpec](#openbaomfamethodspec)
- [OpenBaoMountSpec](#openbaomountspec)
- [OpenBaoNamespaceSpec](#openbaonamespacespec)
- [OpenBaoOIDCAssignmentSpec](#openbaooidcassignmentspec)
- [OpenBaoOIDCClientSpec](#openbaooidcclientspec)
- [OpenBaoOIDCConfigSpec](#openbaooidcconfigspec)
- [OpenBaoOIDCKeySpec](#openbaooidckeyspec)
- [OpenBaoOIDCProviderSpec](#openbaooidcproviderspec)
- [OpenBaoOIDCRoleSpec](#openbaooidcrolespec)
- [OpenBaoOIDCScopeSpec](#openbaooidcscopespec)
- [OpenBaoPasswordPolicySpec](#openbaopasswordpolicyspec)
- [OpenBaoPersonaSpec](#openbaopersonaspec)
- [OpenBaoPluginSpec](#openbaopluginspec)
- [OpenBaoPolicySpec](#openbaopolicyspec)
- [OpenBaoRateLimitQuotaConfigurationSpec](#openbaoratelimitquotaconfigurationspec)
- [OpenBaoRateLimitQuotaSpec](#openbaoratelimitquotaspec)
- [OpenBaoRotationConfigurationSpec](#openbaorotationconfigurationspec)
- [OpenBaoSecretEngineSpec](#openbaosecretenginespec)
- [OpenBaoTokenRoleSpec](#openbaotokenrolespec)
- [OpenBaoUIHeaderSpec](#openbaouiheaderspec)
- [OpenBaoWorkflowSpec](#openbaoworkflowspec)

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


#### OpenBaoEncryptionKeyConfiguration









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoEncryptionKeyConfiguration` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoRotationConfigurationSpec](#openbaorotationconfigurationspec)_ |  |  |  |


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


#### OpenBaoGroupAlias



OpenBaoGroupAlias is the Schema for the openbaogroupaliases API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoGroupAlias` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoGroupAliasSpec](#openbaogroupaliasspec)_ |  |  |  |


#### OpenBaoGroupAliasSpec



OpenBaoGroupAliasSpec defines an alias that maps an authentication mount
identity to an OpenBao identity group.



_Appears in:_
- [OpenBaoGroupAlias](#openbaogroupalias)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `groupRef` _[OpenBaoGroupReference](#openbaogroupreference)_ | GroupRef selects the OpenBaoGroup receiving this alias in the same namespace. |  |  |
| `mountAccessor` _string_ | MountAccessor is the OpenBao auth-method mount accessor for this alias. |  | MinLength: 1 <br /> |
| `name` _string_ | Name is the group name presented by the authentication method. |  | MinLength: 1 <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external alias is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external alias is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |


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
- [OpenBaoGroupAliasSpec](#openbaogroupaliasspec)
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


#### OpenBaoKeyringRotationConfiguration









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoKeyringRotationConfiguration` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoRotationConfigurationSpec](#openbaorotationconfigurationspec)_ |  |  |  |


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
| `audience` _string_ | Audience restricts login JWTs to the configured Kubernetes audience. |  | Optional: \{\} <br /> |
| `tokenType` _string_ | TokenType selects the type of token issued by the role. |  | Enum: [service batch] <br />Optional: \{\} <br /> |
| `tokenNumUses` _integer_ | TokenNumUses limits how many times an issued token may be used. Zero means unlimited. |  | Optional: \{\} <br /> |
| `tokenNoDefaultPolicy` _boolean_ | TokenNoDefaultPolicy prevents OpenBao from adding the default policy to issued tokens. |  | Optional: \{\} <br /> |
| `tokenExplicitMaxTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenExplicitMaxTTL gives issued tokens an explicit maximum TTL. |  | Optional: \{\} <br /> |
| `tokenBoundCIDRs` _string array_ | TokenBoundCIDRs restricts issued tokens to these client address ranges. |  | Optional: \{\} <br /> |


#### OpenBaoLogger









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoLogger` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoLoggerSpec](#openbaologgerspec)_ |  |  |  |


#### OpenBaoLoggerSpec



OpenBaoLoggerSpec configures OpenBao logger verbosity. An empty Name targets
the global logger; a non-empty Name targets one named subsystem logger.



_Appears in:_
- [OpenBaoLogger](#openbaologger)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `name` _string_ |  |  |  |
| `level` _string_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoMFALoginEnforcement









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoMFALoginEnforcement` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoMFALoginEnforcementSpec](#openbaomfaloginenforcementspec)_ |  |  |  |


#### OpenBaoMFALoginEnforcementSpec



OpenBaoMFALoginEnforcementSpec configures which MFA methods are enforced.



_Appears in:_
- [OpenBaoMFALoginEnforcement](#openbaomfaloginenforcement)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `name` _string_ |  |  |  |
| `authMethodAccessors` _string array_ |  |  |  |
| `authMethodTypes` _string array_ |  |  |  |
| `identityEntityIDs` _string array_ |  |  |  |
| `identityGroupIDs` _string array_ |  |  |  |
| `mfaMethodIDs` _string array_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoMFAMethod









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoMFAMethod` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoMFAMethodSpec](#openbaomfamethodspec)_ |  |  |  |


#### OpenBaoMFAMethodSpec



OpenBaoMFAMethodSpec configures an MFA provider. Secret-bearing provider
values are read from same-namespace Secrets and are never written to status.



_Appears in:_
- [OpenBaoMFAMethod](#openbaomfamethod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `type` _[OpenBaoMFAMethodType](#openbaomfamethodtype)_ |  |  | Enum: [duo okta pingid totp] <br /> |
| `methodID` _string_ |  |  |  |
| `methodName` _string_ |  |  |  |
| `usernameFormat` _string_ |  |  |  |
| `apiHostname` _string_ | Duo configuration. |  |  |
| `integrationKey` _string_ |  |  |  |
| `secretKeyRef` _[SecretKeyReference](#secretkeyreference)_ |  |  |  |
| `pushInfo` _string_ |  |  |  |
| `usePasscode` _boolean_ |  |  |  |
| `apiTokenRef` _[SecretKeyReference](#secretkeyreference)_ | Okta configuration. |  |  |
| `baseURL` _string_ |  |  |  |
| `orgName` _string_ |  |  |  |
| `primaryEmail` _boolean_ |  |  |  |
| `production` _boolean_ |  |  |  |
| `settingsFileRef` _[SecretKeyReference](#secretkeyreference)_ | PingID configuration. |  |  |
| `algorithm` _string_ | TOTP configuration. TOTP secret generation is intentionally not part of<br />this resource; these fields configure only the durable method settings. |  |  |
| `digits` _integer_ |  |  |  |
| `issuer` _string_ |  |  |  |
| `keySize` _integer_ |  |  |  |
| `maxValidationAttempts` _integer_ |  |  |  |
| `period` _integer_ |  |  |  |
| `qrSize` _integer_ |  |  |  |
| `skew` _integer_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoMFAMethodType

_Underlying type:_ _string_

OpenBaoMFAMethodType identifies the native OpenBao MFA provider.



_Appears in:_
- [OpenBaoMFAMethodSpec](#openbaomfamethodspec)

| Field | Description |
| --- | --- |
| `duo` |  |
| `okta` |  |
| `pingid` |  |
| `totp` |  |


#### OpenBaoMountSpec



OpenBaoMountSpec is shared by auth-method and secret-engine resources.



_Appears in:_
- [OpenBaoAuthMethodSpec](#openbaoauthmethodspec)
- [OpenBaoSecretEngineSpec](#openbaosecretenginespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `path` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `description` _string_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `local` _boolean_ |  |  |  |
| `sealWrap` _boolean_ |  |  |  |
| `externalEntropyAccess` _boolean_ |  |  |  |
| `pluginName` _string_ |  |  |  |
| `pluginVersion` _string_ |  |  |  |
| `options` _object (keys:string, values:string)_ |  |  |  |
| `config` _object (keys:string, values:string)_ |  |  |  |
| `defaultLeaseTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `maxLeaseTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `listingVisibility` _string_ |  |  |  |
| `tokenType` _string_ |  |  |  |
| `passthroughRequestHeaders` _string array_ |  |  |  |
| `allowedResponseHeaders` _string array_ |  |  |  |


#### OpenBaoNamespace









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoNamespace` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoNamespaceSpec](#openbaonamespacespec)_ |  |  |  |


#### OpenBaoNamespaceSpec



OpenBaoNamespaceSpec manages a child OpenBao namespace inside the connection's namespace.



_Appears in:_
- [OpenBaoNamespace](#openbaonamespace)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `path` _string_ |  |  |  |
| `customMetadata` _object (keys:string, values:string)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoOIDCAssignment









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCAssignment` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCAssignmentSpec](#openbaooidcassignmentspec)_ |  |  |  |


#### OpenBaoOIDCAssignmentSpec



OpenBaoOIDCAssignmentSpec assigns entities and groups to an OIDC assignment.



_Appears in:_
- [OpenBaoOIDCAssignment](#openbaooidcassignment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `entityIDs` _string array_ |  |  |  |
| `groupIDs` _string array_ |  |  |  |


#### OpenBaoOIDCClient









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCClient` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCClientSpec](#openbaooidcclientspec)_ |  |  |  |


#### OpenBaoOIDCClientSpec



OpenBaoOIDCClientSpec configures an OpenBao OIDC client.



_Appears in:_
- [OpenBaoOIDCClient](#openbaooidcclient)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `accessTokenTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `assignments` _string array_ |  |  |  |
| `authorizationCode` _boolean_ |  |  |  |
| `clientCredentials` _boolean_ |  |  |  |
| `clientType` _string_ |  |  |  |
| `idTokenTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `key` _string_ |  |  |  |
| `redirectURIs` _string array_ |  |  |  |


#### OpenBaoOIDCConfig









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCConfig` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCConfigSpec](#openbaooidcconfigspec)_ |  |  |  |


#### OpenBaoOIDCConfigSpec



OpenBaoOIDCConfigSpec configures the issuer for OpenBao's identity OIDC provider.



_Appears in:_
- [OpenBaoOIDCConfig](#openbaooidcconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `issuer` _string_ |  |  |  |


#### OpenBaoOIDCKey









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCKey` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCKeySpec](#openbaooidckeyspec)_ |  |  |  |


#### OpenBaoOIDCKeySpec



OpenBaoOIDCKeySpec configures an OpenBao OIDC signing key.



_Appears in:_
- [OpenBaoOIDCKey](#openbaooidckey)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `algorithm` _string_ |  |  |  |
| `allowedClientIDs` _string array_ |  |  |  |
| `rotationPeriod` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `verificationTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoOIDCProvider









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCProvider` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCProviderSpec](#openbaooidcproviderspec)_ |  |  |  |


#### OpenBaoOIDCProviderSpec



OpenBaoOIDCProviderSpec configures an OpenBao OIDC provider.



_Appears in:_
- [OpenBaoOIDCProvider](#openbaooidcprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `allowedClientIDs` _string array_ |  |  |  |
| `issuer` _string_ |  |  |  |
| `scopesSupported` _string array_ |  |  |  |


#### OpenBaoOIDCRole









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCRole` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCRoleSpec](#openbaooidcrolespec)_ |  |  |  |


#### OpenBaoOIDCRoleSpec



OpenBaoOIDCRoleSpec configures an OpenBao identity OIDC role.



_Appears in:_
- [OpenBaoOIDCRole](#openbaooidcrole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `clientID` _string_ |  |  |  |
| `key` _string_ |  |  |  |
| `template` _string_ |  |  |  |
| `ttl` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoOIDCScope









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoOIDCScope` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoOIDCScopeSpec](#openbaooidcscopespec)_ |  |  |  |


#### OpenBaoOIDCScopeSpec



OpenBaoOIDCScopeSpec configures an OpenBao OIDC scope.



_Appears in:_
- [OpenBaoOIDCScope](#openbaooidcscope)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `description` _string_ |  |  |  |
| `template` _string_ |  |  |  |


#### OpenBaoPasswordPolicy



OpenBaoPasswordPolicy is the Schema for the openbaopasswordpolicies API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoPasswordPolicy` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoPasswordPolicySpec](#openbaopasswordpolicyspec)_ |  |  |  |


#### OpenBaoPasswordPolicySpec



OpenBaoPasswordPolicySpec defines a password-generation policy in OpenBao.



_Appears in:_
- [OpenBaoPasswordPolicy](#openbaopasswordpolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `rules` _string_ | Rules is the OpenBao password policy document in HCL or JSON syntax. |  | MinLength: 1 <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external policy is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external policy is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |


#### OpenBaoPersona









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoPersona` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoPersonaSpec](#openbaopersonaspec)_ |  |  |  |


#### OpenBaoPersonaSpec



OpenBaoPersonaSpec configures a durable OpenBao identity persona.



_Appears in:_
- [OpenBaoPersona](#openbaopersona)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `name` _string_ |  |  |  |
| `entityID` _string_ |  |  |  |
| `mountAccessor` _string_ |  |  |  |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoPlugin









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoPlugin` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoPluginSpec](#openbaopluginspec)_ |  |  |  |


#### OpenBaoPluginSpec



OpenBaoPluginSpec registers a plugin that is already present in OpenBao's plugin directory.



_Appears in:_
- [OpenBaoPlugin](#openbaoplugin)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `name` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `command` _string_ |  |  |  |
| `args` _string array_ |  |  |  |
| `env` _string array_ |  |  |  |
| `sha256` _string_ |  |  |  |
| `version` _string_ |  |  |  |
| `oci` _boolean_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


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


#### OpenBaoRateLimitQuota









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoRateLimitQuota` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoRateLimitQuotaSpec](#openbaoratelimitquotaspec)_ |  |  |  |


#### OpenBaoRateLimitQuotaConfiguration









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoRateLimitQuotaConfiguration` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoRateLimitQuotaConfigurationSpec](#openbaoratelimitquotaconfigurationspec)_ |  |  |  |


#### OpenBaoRateLimitQuotaConfigurationSpec



OpenBaoRateLimitQuotaConfigurationSpec configures global quota behavior.



_Appears in:_
- [OpenBaoRateLimitQuotaConfiguration](#openbaoratelimitquotaconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `enableRateLimitAuditLogging` _boolean_ |  |  |  |
| `enableRateLimitResponseHeaders` _boolean_ |  |  |  |
| `rateLimitExemptPaths` _string array_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoRateLimitQuotaSpec



OpenBaoRateLimitQuotaSpec configures a rate-limit quota.



_Appears in:_
- [OpenBaoRateLimitQuota](#openbaoratelimitquota)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `type` _string_ |  |  |  |
| `path` _string_ |  |  |  |
| `role` _string_ |  |  |  |
| `rate` _string_ | Rate is the positive request rate. It is encoded as a string to preserve<br />exact decimal values across Kubernetes clients and CRD serializers. |  |  |
| `interval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `blockInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `inheritable` _boolean_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoRotationConfigurationSpec



OpenBaoRotationConfigurationSpec configures automatic encryption-key
rotation. It does not perform an immediate rotation.



_Appears in:_
- [OpenBaoEncryptionKeyConfiguration](#openbaoencryptionkeyconfiguration)
- [OpenBaoKeyringRotationConfiguration](#openbaokeyringrotationconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `enabled` _boolean_ |  |  |  |
| `interval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `maxOperations` _integer_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoSecretEngine









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoSecretEngine` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoSecretEngineSpec](#openbaosecretenginespec)_ |  |  |  |


#### OpenBaoSecretEngineSpec



OpenBaoSecretEngineSpec enables and tunes an OpenBao secret engine.



_Appears in:_
- [OpenBaoSecretEngine](#openbaosecretengine)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `path` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `description` _string_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `local` _boolean_ |  |  |  |
| `sealWrap` _boolean_ |  |  |  |
| `externalEntropyAccess` _boolean_ |  |  |  |
| `pluginName` _string_ |  |  |  |
| `pluginVersion` _string_ |  |  |  |
| `options` _object (keys:string, values:string)_ |  |  |  |
| `config` _object (keys:string, values:string)_ |  |  |  |
| `defaultLeaseTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `maxLeaseTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |
| `listingVisibility` _string_ |  |  |  |
| `tokenType` _string_ |  |  |  |
| `passthroughRequestHeaders` _string array_ |  |  |  |
| `allowedResponseHeaders` _string array_ |  |  |  |


#### OpenBaoTokenRole



OpenBaoTokenRole is the Schema for the openbaotokenroles API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoTokenRole` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoTokenRoleSpec](#openbaotokenrolespec)_ |  |  |  |


#### OpenBaoTokenRoleSpec



OpenBaoTokenRoleSpec defines a role in OpenBao's built-in token auth method.



_Appears in:_
- [OpenBaoTokenRole](#openbaotokenrole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ | ConnectionRef selects the OpenBao API connection in the same namespace. |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls how an external role is acquired. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external role is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | DriftDetectionInterval controls periodic checks for changes made outside Kubernetes.<br />A zero duration disables periodic checks. When omitted, the operator default is used. |  | Optional: \{\} <br /> |
| `allowedEntityAliases` _string array_ | AllowedEntityAliases limits entity aliases that may be attached to tokens from this role. |  | Optional: \{\} <br /> |
| `allowedPolicies` _string array_ | AllowedPolicies limits policies that callers may request for generated tokens. |  | Optional: \{\} <br /> |
| `allowedPoliciesGlob` _string array_ | AllowedPoliciesGlob limits policies using OpenBao glob patterns. |  | Optional: \{\} <br /> |
| `disallowedPolicies` _string array_ | DisallowedPolicies rejects requested policies in this set. |  | Optional: \{\} <br /> |
| `disallowedPoliciesGlob` _string array_ | DisallowedPoliciesGlob rejects requested policies matching these globs. |  | Optional: \{\} <br /> |
| `tokenBoundCIDRs` _string array_ | TokenBoundCIDRs restricts issued tokens to these client address ranges. |  | Optional: \{\} <br /> |
| `tokenExplicitMaxTTL` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenExplicitMaxTTL gives issued tokens an explicit maximum TTL. |  | Optional: \{\} <br /> |
| `tokenPeriod` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ | TokenPeriod gives issued tokens a fixed renewable period. |  | Optional: \{\} <br /> |
| `tokenNumUses` _integer_ | TokenNumUses limits how many times an issued token may be used. Zero means unlimited. |  | Optional: \{\} <br /> |
| `tokenNoDefaultPolicy` _boolean_ | TokenNoDefaultPolicy prevents OpenBao from adding the default policy. |  | Optional: \{\} <br /> |
| `tokenType` _string_ | TokenType selects the type of token generated by the role. |  | Enum: [service batch] <br />Optional: \{\} <br /> |
| `orphan` _boolean_ | Orphan controls whether generated tokens have no parent token. |  | Optional: \{\} <br /> |
| `renewable` _boolean_ | Renewable controls whether generated tokens can be renewed. |  | Optional: \{\} <br /> |
| `pathSuffix` _string_ | PathSuffix adds a suffix to generated token paths for targeted revocation. |  | Optional: \{\} <br /> |


#### OpenBaoUIHeader









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoUIHeader` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoUIHeaderSpec](#openbaouiheaderspec)_ |  |  |  |


#### OpenBaoUIHeaderSpec



OpenBaoUIHeaderSpec configures one OpenBao UI response header.



_Appears in:_
- [OpenBaoUIHeader](#openbaouiheader)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `header` _string_ |  |  |  |
| `values` _string array_ |  |  |  |
| `multivalue` _boolean_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### OpenBaoWorkflow









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openbao.openbao-operator.io/v1alpha1` | | |
| `kind` _string_ | `OpenBaoWorkflow` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OpenBaoWorkflowSpec](#openbaoworkflowspec)_ |  |  |  |


#### OpenBaoWorkflowSpec



OpenBaoWorkflowSpec manages a stored OpenBao workflow definition.



_Appears in:_
- [OpenBaoWorkflow](#openbaoworkflow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[OpenBaoConnectionReference](#openbaoconnectionreference)_ |  |  |  |
| `path` _string_ |  |  |  |
| `workflow` _string_ |  |  |  |
| `description` _string_ |  |  |  |
| `allowUnauthenticated` _boolean_ |  |  |  |
| `cas` _integer_ |  |  |  |
| `casRequired` _boolean_ |  |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ |  |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ |  |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta)_ |  |  |  |


#### SecretKeyReference



SecretKeyReference identifies a key in a same-namespace Secret.



_Appears in:_
- [AppRoleAuthSpec](#approleauthspec)
- [OpenBaoConnectionSpec](#openbaoconnectionspec)
- [OpenBaoMFAMethodSpec](#openbaomfamethodspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret resource name. |  | MinLength: 1 <br /> |
| `key` _string_ | Key is the Secret data key. Defaults depend on the referencing field:<br />token for token references, role-id for AppRole role IDs, secret-id for<br />AppRole Secret IDs, and ca.crt for CA bundle references. |  | Optional: \{\} <br /> |
