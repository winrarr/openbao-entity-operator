# Verification guide

## Canonical checks

| Command | What it proves | Limits |
| --- | --- | --- |
| `make format-check` | Go sources are formatted | Does not assess behavior |
| `make shell-check` | Repository shell scripts parse successfully | Does not assess shell behavior against a cluster |
| `make manifests generate` | CRDs, RBAC, and deepcopy output can be regenerated | Does not prove generated output was committed |
| `make verify-generated` | Tracked generated output has no regeneration diff | Requires a Git checkout and compares only the protected paths |
| `make test` | Typed OpenBao HTTP contracts, authentication login/renew/re-login, policy and role lifecycle, system-resource lifecycle, and identity reconciliation transitions pass unit tests | Does not exercise a live Kubernetes API server or OpenBao |
| `make lint-config lint` | Linter configuration and source checks pass | Linter findings are not a substitute for runtime tests |
| `make openapi-check` | The checked-in OpenBao reference is valid and contains the selected durable OpenBao API families used here | The reference is version-specific and dynamic |
| `make kustomize-build` | The default installation manifests render | Does not apply them to a cluster |
| `make helm-lint helm-template` | The Helm chart values validate and the chart renders | Does not install the chart |
| `make helm-package` | Packages the validated Helm chart, including committed CRDs, into `dist/` | Does not publish the package |
| `make docs-build` | The generated CRD reference is current and the documentation site passes strict validation | Does not publish the site locally |
| `make check` | Runs the complete local foundation suite | Does not start external services |
| `make kind-e2e` | Installs the operator from the Helm chart with a namespace scope, verifies scoped CRD/RBAC presence and an ignored resource outside the scope, authenticates through real OpenBao Kubernetes Auth, AppRole, and token connections, exercises one live policy/Kubernetes Auth role/entity/alias/group-membership graph including bound and unbound workload login, verifies two-tenant role isolation, and proves Delete-policy cleanup resumes after credential loss | Uses a single in-memory OpenBao dev server; controller branches such as adoption, drift, conflicts, and most deletion behavior remain covered by unit and HTTP contract tests |
| `make kind-e2e-resilience` | Verifies the operator authenticates with a non-root token over TLS, reconciles a policy, recovers reconciliation after the OpenBao Pod restarts, and recovers after token A is revoked and the Kubernetes credential Secret is rotated to token B | Uses a disposable one-node persistent OpenBao fixture only as external plumbing; does not prove OpenBao storage, Raft, TLS, ACL, HA, or backup correctness |
| `make kind-e2e-clean` | Removes only the named live E2E namespaces after deleting resources in dependency order | Does not remove external OpenBao objects left by a failed cleanup |
| `make kind-e2e-resilience-clean` | Removes the retained resilience test and fixture namespaces, then restores the default cluster-wide Helm installation | Deletes only the fixed resilience fixture in the named Kind cluster |
| `make kind-down` | Removes only the named disposable Kind cluster | Deletes local OpenBao data and test resources |

The unit tests use `httptest.Server` for HTTP contracts and controller-runtime's fake Kubernetes client with injected OpenBao clients for reconciliation. The Kind workflow adds focused live Kubernetes/OpenBao evidence without duplicating the controller state machine in raw shell.

## Evidence rules

When changing OpenBao API behavior, add or update a focused client test and record external API facts in `docs/research/`. When changing API markers, regenerate and inspect CRD and RBAC diffs. Live tests must use the isolated Kind/OpenBao environment; never point them at a shared or production cluster.
