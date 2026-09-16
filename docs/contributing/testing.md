# Testing

The repository has three verification layers:

- `make test` runs unit, fake-client reconciliation, and HTTP contract tests without external services.
- `make check` adds generation, formatting, vet, lint, OpenAPI validation, and Kustomize rendering.
- `make kind-e2e` runs the live OpenBao and Kubernetes scenarios in an isolated Kind cluster.

The `Live Kind tests` workflow runs the same `make kind-e2e` target on pushes and pull requests. The Cilium variant remains a local opt-in because the default suite is focused on operator and OpenBao behavior rather than CNI policy enforcement.

The `Tests` and `Lint` workflows run the non-live checks on pushes and pull requests. The `Docs` workflow generates the CRD reference and performs the strict site build; pushes to `main` publish the resulting site to GitHub Pages. Workflows cancel superseded runs for the same branch or pull request.

The live workflow is intentionally separate from `make check` because it downloads images, starts a cluster, and uses an in-memory OpenBao instance. A failed run retains its test namespace for inspection; clean it up with `make kind-e2e-clean`, or use `make kind-down` to remove the entire named disposable cluster.

When changing an OpenBao endpoint, update the typed client, add an HTTP contract test, and run the live workflow. When changing API markers or controller permissions, run `make manifests generate` and `make verify-generated`.
