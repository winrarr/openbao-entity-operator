# Examples

These examples are intentionally small and build on the same-namespace
connection model used by the operator.

## Connection and identity

The [connection and identity example](connection-and-identity.md) creates a
token Secret, connection, policy, entity, alias, group, and two membership
claims in dependency order.

## Repository samples

The `config/samples/` directory contains Kubebuilder-compatible manifests for
each API type. They are useful as schema-shaped starting points, but replace
addresses, Secret names, auth mount accessors, and policy documents with values
appropriate for the target OpenBao instance.
