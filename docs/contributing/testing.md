# Testing

The repository has three verification layers with deliberately different jobs:

- `make test` runs unit, fake-client reconciliation, and HTTP contract tests without external services. This is where almost all controller logic belongs: creation and adoption policy, drift, status, finalizers, dependency failures, and error handling.
- `make samples-check` renders every checked-in sample bundle, and `make conformance` combines that check with the full local test suite.
- `make check` verifies generated output, formatting, vet, lint, Helm chart lint/rendering, OpenAPI validation, sample rendering, and strict docs validation.
- `make kind-e2e` runs a focused live smoke/integration path in an isolated Kind cluster: scoped Helm installation, CRDs/RBAC, a negative watch-scope check, Kubernetes Auth, AppRole, and one successful policy/entity/alias/group-membership graph.

The live script is intentionally not a second controller test suite. Add a new
E2E scenario only when it proves Kubernetes/OpenBao wiring or an external
behavior that fake clients and HTTP contract tests cannot observe. See
[Decision 0005](../decisions/0005-test-pyramid.md) for the accepted test
boundary.

The conformance suite uses shared fake OpenBao clients and table-driven cases
to keep the broad resource surface maintainable. When adding a resource,
prefer a typed HTTP contract test, a focused controller test for ownership and
status semantics, and a credential-free sample before expanding the live Kind
workflow.

The `Live Kind tests` workflow runs the same `make kind-e2e` target on pushes and pull requests. It installs the operator through the committed Helm chart. The Cilium variant remains a local opt-in because the default suite is focused on operator and OpenBao behavior rather than CNI policy enforcement.

`make kind-e2e-resilience` is an opt-in recovery test, not a second copy of
the normal regression suite. Run it when changing connection authentication,
credential Secret rotation, token renewal or re-login, TLS handling, client
recovery, or restart behavior. It remains separate because the persistent TLS
fixture is slower and more operationally involved than the normal live test;
its assertions are about operator recovery, not OpenBao storage or Raft.

The `Tests` and `Lint` workflows run the non-live checks on pushes and pull requests. The `Docs` workflow generates the CRD reference and performs the strict site build; pushes to `main` publish the resulting site to GitHub Pages. Workflows cancel superseded runs for the same branch or pull request.

The live workflow is intentionally separate from `make check` because it downloads images, starts a cluster, and uses an in-memory OpenBao instance. A failed run retains its test namespaces for inspection; clean them up with `make kind-e2e-clean`, or use `make kind-down` to remove the entire named disposable cluster.

When changing an OpenBao endpoint or authentication flow, update the typed client, add an HTTP contract test, record the external contract in `docs/research/`, and run the live workflow. When changing API markers or controller permissions, run `make manifests generate` and `make verify-generated`.
