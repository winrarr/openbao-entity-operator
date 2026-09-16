# OpenBaoPolicy

`OpenBaoPolicy` manages one OpenBao ACL policy. The Kubernetes
`metadata.name` is used as the OpenBao policy name, and `spec.rules` is the raw
HCL or JSON policy document sent to OpenBao.

## Example

```yaml
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
```

The operator reads the policy through OpenBao's ACL policy endpoint, creates it
when missing, and writes the exact desired document when it drifts. The
document remains in `spec`; status contains only the observed policy name,
version, and SHA-256 rules hash.

## Existing policies

The default `creationPolicy: Create` protects an existing unacquired policy
from accidental overwrite. Choose one of these policies deliberately:

- `Create` — create a missing policy and reject an existing unacquired policy;
- `Adopt` — manage an existing policy and reject a missing policy; or
- `CreateOrAdopt` — create when missing and adopt when present.

After adoption, later reconciliations recognize the policy as acquired by this
resource. Adoption does not change the policy's name.

## Deletion and drift

Policies default to `deletionPolicy: Orphan`. Select `Delete` to remove the
OpenBao policy when the Kubernetes object is deleted. Periodic drift checks use
`driftDetectionInterval`; a successful reconciliation updates `status.version`
and `status.rulesHash`.

```sh
kubectl get openbaopolicy/payments
kubectl get openbaopolicy/payments -o jsonpath='{.status.rulesHash}{"\n"}'
kubectl describe openbaopolicy/payments
```

See the [generated OpenBaoPolicy schema](../reference/api.md#openbaopolicy)
for exact validation and defaults.
