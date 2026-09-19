# Durable configuration example

This example shows the broad configuration surface without deploying OpenBao.
Create the OpenBao instance and its initial connection credential separately,
then apply the connection and the resources that target it.

The repository's [configuration sample](../../config/samples/openbao_v1alpha1_openbao_system_configuration.yaml)
covers mounts, namespaces, audit, quotas, workflows, plugins, and system
settings. The [identity sample](../../config/samples/openbao_v1alpha1_openbao_identity_configuration.yaml)
and [OIDC sample](../../config/samples/openbao_v1alpha1_openbao_oidc_configuration.yaml)
cover the remaining durable configuration families.

## Apply a focused configuration

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoSecretEngine
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  path: payments
  type: kv
  options:
    version: "2"
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: payments-read
spec:
  connectionRef:
    name: openbao
  rules: |
    path "payments/data/*" {
      capabilities = ["read"]
    }
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoTokenRole
metadata:
  name: payments-reader
spec:
  connectionRef:
    name: openbao
  allowedPolicies:
    - payments-read
  tokenType: service
  tokenPeriod: 1h
```

Apply resources in dependency order when one resource refers to another:

```sh
kubectl apply -f connection.yaml
kubectl wait --for=condition=Ready openbaoconnection/openbao --timeout=2m
kubectl apply -f durable-configuration.yaml
kubectl wait --for=condition=Ready openbaosecretengine/payments --timeout=2m
kubectl wait --for=condition=Ready openbaotokenrole/payments-reader --timeout=2m
```

The operator manages configuration records and reports status. It does not
write arbitrary secret-engine data, issue tokens, install plugin binaries, or
execute workflows. Keep `deletionPolicy: Orphan` unless the OpenBao-side
deletion impact is intentional.
