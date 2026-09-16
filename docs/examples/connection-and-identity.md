# Connection and identity example

Apply the objects in dependency order. The token Secret is shown only as an
environment-substituted placeholder; do not commit a real token.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openbao-token
type: Opaque
stringData:
  token: ${OPENBAO_TOKEN}
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao
spec:
  address: https://openbao.example.com:8200
  tokenSecretRef:
    name: openbao-token
    key: token
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  rules: |
    path "secret/data/payments/*" {
      capabilities = ["read"]
    }
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: payments
spec:
  connectionRef:
    name: openbao
  policies:
    - payments
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroup
metadata:
  name: platform
spec:
  connectionRef:
    name: openbao
  policies:
    - payments
---
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

```sh
kubectl apply -f identity.yaml
kubectl wait --for=condition=Ready openbaoconnection/openbao --timeout=2m
kubectl wait --for=condition=Ready openbaopolicy/payments --timeout=2m
kubectl wait --for=condition=Ready openbaoentity/payments --timeout=2m
kubectl wait --for=condition=Ready openbaogroup/platform --timeout=2m
kubectl wait --for=condition=Ready openbaogroupmembership/platform-payments --timeout=2m
```

Add an alias after the entity is ready. The `mountAccessor` must be obtained
from the configured OpenBao auth method:

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
