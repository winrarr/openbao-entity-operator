# Verification guide

## Canonical checks

| Command | What it proves | Limits |
| --- | --- | --- |
| `make format-check` | Go sources are formatted | Does not assess behavior |
| `make shell-check` | Repository shell scripts parse successfully | Does not assess shell behavior against a cluster |
| `make manifests generate` | CRDs, RBAC, and deepcopy output can be regenerated | Does not prove generated output was committed |
| `make verify-generated` | Tracked generated output has no regeneration diff | Requires a Git checkout and compares only the protected paths |
| `make test` | Client HTTP behavior, Kubernetes Auth and AppRole login/renew/re-login, policy lifecycle, and identity reconciliation transitions pass unit tests | Does not exercise a live Kubernetes API server or OpenBao |
| `make lint-config lint` | Linter configuration and source checks pass | Linter findings are not a substitute for runtime tests |
| `make openapi-check` | The checked-in OpenBao reference is valid and contains the entity, alias, group, and ACL policy endpoints used here | The reference is version-specific and dynamic |
| `make kustomize-build` | The default installation manifests render | Does not apply them to a cluster |
| `make helm-lint helm-template` | The Helm chart values validate and the chart renders | Does not install the chart |
| `make helm-package` | Packages the validated Helm chart, including committed CRDs, into `dist/` | Does not publish the package |
| `make docs-build` | The generated CRD reference is current and the documentation site passes strict validation | Does not publish the site locally |
| `make check` | Runs the complete local foundation suite | Does not start external services |
| `make kind-e2e` | Installs the operator from the Helm chart, verifies CRD/RBAC presence, authenticates through real OpenBao Kubernetes Auth and AppRole connections, and exercises one live policy/entity/alias/group-membership graph | Uses a single in-memory OpenBao dev server; controller branches such as adoption, drift, conflicts, deletion policies, and dependency loss are covered by unit and HTTP contract tests |
| `make kind-e2e-clean` | Removes only the named live E2E test namespace after deleting resources in dependency order | Does not remove external OpenBao objects left by a failed cleanup |
| `make kind-down` | Removes only the named disposable Kind cluster | Deletes local OpenBao data and test resources |

The unit tests use `httptest.Server` for HTTP contracts and controller-runtime's fake Kubernetes client with injected OpenBao clients for reconciliation. The Kind workflow adds focused live Kubernetes/OpenBao evidence without duplicating the controller state machine in raw shell.

## Evidence rules

When changing OpenBao API behavior, add or update a focused client test and record external API facts in `docs/research/`. When changing API markers, regenerate and inspect CRD and RBAC diffs. Live tests must use the isolated Kind/OpenBao environment; never point them at a shared or production cluster.
