# Reference

Use the reference section for cross-cutting behavior and exact schema details:

- [Resource index](resources.md) links practical guides to generated API
  sections.
- [Connection patterns](connection-patterns.md) explains authentication,
  namespace routing, and CA material.
- [Deletion and ownership](deletion-and-ownership.md) explains adoption,
  orphaning, external deletion, and finalizers.
- [Status and conditions](status-and-conditions.md) explains what controllers
  publish while reconciling.
- [Troubleshooting](troubleshooting.md) groups common failure modes and
  recovery commands.
- [Operator configuration](operator-configuration.md) documents the chart and
  runtime settings that affect installation.
- [Generated API reference](api.md) is the exact schema generated from the Go
  API types.

The generated page is refreshed with:

```sh
make generate-api-reference
```

It is checked by `make check` and `make verify-generated`; edit API comments and
Kubebuilder markers rather than editing the generated page directly.
