# Testing

The repository has three verification layers with deliberately different jobs:

- `make test` runs unit, fake-client reconciliation, and HTTP contract tests without external services. This is where almost all controller logic belongs: creation and adoption policy, drift, status, finalizers, dependency failures, and error handling.
- `make check` adds generation, formatting, vet, lint, Helm chart lint/rendering, OpenAPI validation, and Kustomize rendering.
- `make kind-e2e` runs a focused live smoke/integration path in an isolated Kind cluster: scoped Helm installation, CRDs/RBAC, a negative watch-scope check, Kubernetes Auth, AppRole, and one successful policy/entity/alias/group-membership graph.

The live script is intentionally not a second controller test suite. Add a new
E2E scenario only when it proves Kubernetes/OpenBao wiring or an external
behavior that fake clients and HTTP contract tests cannot observe. See
[Decision 0005](../decisions/0005-test-pyramid.md) for the accepted test
boundary.

The `Live Kind tests` workflow runs the same `make kind-e2e` target on pushes and pull requests. It installs the operator through the committed Helm chart. The Cilium variant remains a local opt-in because the default suite is focused on operator and OpenBao behavior rather than CNI policy enforcement.

The `Tests` and `Lint` workflows run the non-live checks on pushes and pull requests. The `Docs` workflow generates the CRD reference and performs the strict site build; pushes to `main` publish the resulting site to GitHub Pages. Workflows cancel superseded runs for the same branch or pull request.

The live workflow is intentionally separate from `make check` because it downloads images, starts a cluster, and uses an in-memory OpenBao instance. A failed run retains its test namespaces for inspection; clean them up with `make kind-e2e-clean`, or use `make kind-down` to remove the entire named disposable cluster.

When changing an OpenBao endpoint or authentication flow, update the typed client, add an HTTP contract test, record the external contract in `docs/research/`, and run the live workflow. When changing API markers or controller permissions, run `make manifests generate` and `make verify-generated`.
