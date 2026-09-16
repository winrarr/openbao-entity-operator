# OpenBao AppRole

`OpenBaoConnection` supports AppRole authentication when Kubernetes Auth is not
available or is not the right trust boundary. The operator reads the AppRole
role ID and Secret ID from two same-namespace Kubernetes Secrets, logs in through
OpenBao, and uses the returned token for dependent resources.

## OpenBao-side setup

An OpenBao administrator must enable the AppRole auth method and configure a
role before the operator can use it. The role must grant only the identity,
policy, and token operations needed by the resources that will use the
connection. The operator does not enable the auth method, create the role, or
issue Secret IDs.

The default login endpoint is `auth/approle/login`. Set
`spec.appRole.mountPath` when the method is mounted elsewhere; the value is
relative to `auth/` and must not include that prefix.

## Kubernetes-side setup

Store the two AppRole credentials in Secrets in the same namespace as the
connection. The default keys are `role-id` and `secret-id`; use `key` to select
different keys.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openbao-approle-role
stringData:
  role-id: ${OPENBAO_APPROLE_ROLE_ID}
---
apiVersion: v1
kind: Secret
metadata:
  name: openbao-approle-secret
stringData:
  secret-id: ${OPENBAO_APPROLE_SECRET_ID}
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao-approle
spec:
  address: https://openbao.example.com:8200
  appRole:
    mountPath: approle
    roleIDSecretRef:
      name: openbao-approle-role
    secretIDSecretRef:
      name: openbao-approle-secret
```

Do not commit the substituted credential values to Git. Kubernetes RBAC should
allow the operator to read only the namespaces and Secrets required by its
deployment model.

## Credential lifecycle

The operator caches the short-lived OpenBao token in memory so renewable leases
can be renewed across reconciliations. If renewal fails or OpenBao rejects the
token, it reads both credential Secrets again and performs one fresh AppRole
login. Updating either Secret therefore takes effect on the next login without
restarting the operator.

The operator does not rotate or revoke AppRole Secret IDs. Choose an OpenBao
Secret ID TTL and use-count that leave enough time for the external credential
rotation process to publish a replacement Secret before the current token must
log in again. Credential values are never written to status or logs.

See the [OpenBao AppRole documentation](https://openbao.org/docs/auth/approle/)
for the auth method's configuration and credential constraints.
