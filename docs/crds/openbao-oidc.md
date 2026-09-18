# OpenBao OIDC resources

OpenBao OIDC resources configure the durable records used by OpenBao's
identity OIDC provider. They operate an existing OpenBao instance and do not
run an OIDC login flow or copy generated client secrets into Kubernetes.

## OpenBaoOIDCConfig

`OpenBaoOIDCConfig` manages the provider issuer configuration. Treat it as a
singleton per connection and protect it with an OpenBao policy that is owned by
the platform.

```yaml
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoOIDCConfig
metadata:
  name: issuer
spec:
  connectionRef:
    name: openbao
  issuer: https://openbao.example.com:8200
```

## OpenBaoOIDCProvider

`OpenBaoOIDCProvider` declares the provider's issuer-facing metadata and
allowed client IDs.

## OpenBaoOIDCKey

`OpenBaoOIDCKey` declares a signing key's algorithm, allowed clients, rotation
period, and verification TTL. Key material remains managed by OpenBao.

## OpenBaoOIDCRole

`OpenBaoOIDCRole` maps an OpenBao identity OIDC role to a key and client. The
template is stored as configuration; the operator does not issue tokens.

## OpenBaoOIDCScope

`OpenBaoOIDCScope` stores a reusable OIDC scope template and description.

## OpenBaoOIDCAssignment

`OpenBaoOIDCAssignment` associates a client with named scopes. Assignments
are ordinary durable configuration and are drift-corrected like the other
resources.

## OpenBaoOIDCClient

`OpenBaoOIDCClient` configures a client, redirect URIs, grant capabilities,
token lifetimes, and assignments. OpenBao may generate a client secret; that
secret is intentionally not returned by this operator or written to a
Kubernetes Secret. Use a separate credential-delivery workflow if an
application needs the secret.

For all OIDC resources, use `creationPolicy: Adopt` when bringing an existing
OpenBao record under management and leave `deletionPolicy` at `Orphan` until
the external impact of deletion is understood.
