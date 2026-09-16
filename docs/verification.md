# Verification guide

## Canonical checks

| Command | What it proves | Limits |
| --- | --- | --- |
| `make format-check` | Go sources are formatted | Does not assess behavior |
| `make shell-check` | Repository shell scripts parse successfully | Does not assess shell behavior against a cluster |
| `make manifests generate` | CRDs, RBAC, and deepcopy output can be regenerated | Does not prove generated output was committed |
| `make verify-generated` | Tracked generated output has no regeneration diff | Requires a Git checkout and compares only the protected paths |
| `make test` | Client HTTP behavior and reconciliation transitions pass unit tests | Does not exercise a live Kubernetes API server or OpenBao |
| `make lint-config lint` | Linter configuration and source checks pass | Linter findings are not a substitute for runtime tests |
| `make openapi-check` | The checked-in OpenBao reference is valid and contains the entity, alias, and group endpoints used here | The reference is version-specific and dynamic |
| `make kustomize-build` | The default installation manifests render | Does not apply them to a cluster |
| `make docs-build` | The generated CRD reference is current and the documentation site passes strict validation | Does not publish the site locally |
| `make check` | Runs the complete local foundation suite | Does not start external services |
| `make kind-e2e` | Runs live connection, entity, alias, group, and membership lifecycle, drift, adoption, conflict, orphan, delete, and missing-cleanup-dependency scenarios against OpenBao in Kind | Uses a single in-memory OpenBao dev server |
| `make kind-e2e-clean` | Removes only the named live E2E test namespace after deleting resources in dependency order | Does not remove external OpenBao objects left by a failed cleanup |
| `make kind-down` | Removes only the named disposable Kind cluster | Deletes local OpenBao data and test resources |

The unit tests use `httptest.Server` for HTTP contracts and controller-runtime's fake Kubernetes client with injected OpenBao clients for reconciliation. The Kind workflow adds live Kubernetes/OpenBao evidence without making the external cluster part of the ordinary check suite.

## Evidence rules

When changing OpenBao API behavior, add or update a focused client test and record external API facts in `docs/research/`. When changing API markers, regenerate and inspect CRD and RBAC diffs. Live tests must use the isolated Kind/OpenBao environment; never point them at a shared or production cluster.
