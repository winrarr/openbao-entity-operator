# Identity and authentication configuration resources

These resources configure durable OpenBao identity and authentication records
after OpenBao is deployed. They do not issue credentials, copy secret material
to Kubernetes, or perform login and token flows.

## OpenBaoGroupAlias

`OpenBaoGroupAlias` binds an alias name and auth mount accessor to an external
`OpenBaoGroup`:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupAlias
metadata:
  name: platform-from-idp
spec:
  connectionRef:
    name: openbao
  groupRef:
    name: platform
  mountAccessor: auth_oidc_accessor
  name: platform
```

The referenced group must be an external group. The alias is an owned binding;
it does not synchronize group membership from the external identity provider.

## OpenBaoTokenRole

`OpenBaoTokenRole` configures the constraints applied to tokens created through
an OpenBao token role. It never creates a token:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoTokenRole
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  allowedPolicies:
    - payments-read
  tokenType: service
  tokenPeriod: 1h
  tokenNoDefaultPolicy: true
```

Use the connection's OpenBao ACL to limit which roles a tenant may manage.

## OpenBaoAppRole

`OpenBaoAppRole` configures an AppRole role at an existing AppRole mount. It
does not generate, return, or rotate Role IDs or Secret IDs:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoAppRole
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  bindSecretID: true
  tokenPolicies:
    - payments-read
  tokenTTL: 1h
```

Credential issuance and delivery should be handled by a separate, explicitly
designed workflow.

## OpenBaoPasswordPolicy

`OpenBaoPasswordPolicy` stores a named OpenBao password-policy document. The
rules are raw HCL or JSON and are reconciled exactly:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPasswordPolicy
metadata:
  name: enterprise
spec:
  connectionRef:
    name: openbao
  rules: |
    length = 20
    rule "charset" {
      charset = "abcdefghijklmnopqrstuvwxyz"
      min-chars = 2
    }
```

The resource manages the policy definition, not passwords or password
rotation requests.

## OpenBaoPersona

`OpenBaoPersona` manages a durable identity persona by its stable OpenBao ID.
It binds the persona to an entity and, when applicable, an auth mount
accessor. Persona IDs are discovered only for initial adoption; subsequent
reconciliation uses the ID stored in status.

## OpenBaoMFAMethod

`OpenBaoMFAMethod` configures a native OpenBao MFA provider method. The
resource supports Duo, Okta, PingID, and TOTP configuration. Duo, Okta, and
PingID credentials are referenced from same-namespace Secrets:

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoMFAMethod
metadata:
  name: login-totp
spec:
  connectionRef:
    name: openbao
  type: totp
  methodID: login-totp
  methodName: login-totp
  issuer: example
  algorithm: SHA256
  digits: 6
```

Secret values are sent to OpenBao only when the selected provider requires
them. They are never written to status or logs. TOTP secret generation,
validation, and administrative destroy operations are intentionally outside
this declarative resource.

## OpenBaoMFALoginEnforcement

`OpenBaoMFALoginEnforcement` associates one or more configured MFA method IDs
with selected auth methods, entity IDs, or group IDs. It manages the durable
login-enforcement policy and does not perform login or MFA validation.
