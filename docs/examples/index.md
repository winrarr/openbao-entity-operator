# Examples

These examples are intentionally small and build on the same-namespace
connection model used by the operator.

## Connection and identity

The [connection and identity example](connection-and-identity.md) creates a
token Secret, connection, policy, entity, alias, group, and two membership
claims in dependency order.

## Durable configuration

The [durable configuration example](durable-configuration.md) explains how to
configure mounts and OpenBao records after the server exists. The repository
also provides grouped samples for [identity](../../config/samples/openbao_v1alpha1_openbao_identity_configuration.yaml),
[OIDC](../../config/samples/openbao_v1alpha1_openbao_oidc_configuration.yaml),
and [system configuration](../../config/samples/openbao_v1alpha1_openbao_system_configuration.yaml).

## Repository samples

The `config/samples/` directory contains Kubebuilder-compatible manifests for
each API type. They are useful as schema-shaped starting points, but replace
addresses, Secret names, auth mount accessors, and policy documents with values
appropriate for the target OpenBao instance.
