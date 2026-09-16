# OpenBao Entity Operator

OpenBao Entity Operator manages OpenBao ACL policies and identity resources
through Kubernetes custom resources. This site explains how to install the
operator, connect it to OpenBao, declare resources, and operate the resulting
reconciliation loops.

## Start here

- [Quickstart](quickstart.md) — install the chart, create a connection, and
  declare your first policy and entity.
- [Installation](introduction/installation.md) — choose a published chart,
  checkout, or local Kind installation.
- [Concepts](introduction/concepts.md) — understand connections, ownership,
  drift correction, and resource relationships.
- [Resource guides](crds/index.md) — learn the behavior and examples for each
  custom resource.

## Find a task

- [Connection patterns](reference/connection-patterns.md) — token Secrets,
  Kubernetes Auth, OpenBao namespaces, and CA bundles.
- [Deletion and ownership](reference/deletion-and-ownership.md) — decide when
  an external OpenBao object should be deleted.
- [Status and conditions](reference/status-and-conditions.md) — inspect
  readiness, external IDs, policy versions, and failures.
- [Troubleshooting](reference/troubleshooting.md) — recover from common
  connection, adoption, drift, and finalizer problems.
- [Local Kind environment](operations/local-kind.md) — run the complete live
  workflow locally with the default CNI or Cilium.

## Project knowledge

- [Product scope](product.md)
- [Compatibility policy](compatibility.md)
- [Architecture](architecture.md)
- [User stories](user-stories.md)
- [Decisions](decisions/index.md)
- [Backlog](backlog.md)
- [Tech-debt register](tech-debt.md)
- [Verification](verification.md)
- [OpenBao API research](research/2026-09-16-openbao-api.md)

## Documentation model

Hand-written pages explain behavior, examples, operational constraints, and
design rationale. The [generated API reference](reference/api.md) is produced
from the Go API types and Kubebuilder markers and is the authority for exact
field names, defaults, enums, and validation rules.

The published site follows `main`. Build it locally with:

```sh
make docs-build
```

The documentation workflow builds pull requests and publishes successful pushes
to `main` through GitHub Pages.
