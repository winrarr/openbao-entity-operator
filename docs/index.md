# Documentation map

- [Product scope](product.md)
- [User stories and design coverage](user-stories.md)
- [Architecture](architecture.md)
- [Verification](verification.md)
- [Operations](operations/index.md)
- [Contributing](contributing/index.md)
- [Release process](contributing/releases.md)
- [OpenBao API research](research/2026-09-16-openbao-api.md)
- [Decisions](decisions/index.md)
- [Backlog](backlog.md)
- [Tech-debt register](tech-debt.md)

Generated API manifests live under `config/crd/bases/`. The checked-in OpenBao API reference lives at `hack/openbao-openapi.json` and is refreshed only through the documented Make target.

The generated Kubernetes API reference lives at [`reference/api.md`](reference/api.md). The strict site is built with `make docs-build` and published from `main` by the documentation workflow.
