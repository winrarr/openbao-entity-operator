# Project guide

## Orientation

This is a Go 1.27 Kubernetes operator for OpenBao ACL policies, identity entities, and groups. The public API in `api/openbao/v1alpha1` is the source of truth for the namespaced `OpenBaoConnection`, `OpenBaoPolicy`, `OpenBaoEntity`, `OpenBaoEntityAlias`, `OpenBaoGroup`, and `OpenBaoGroupMembership` CRDs.

- `OpenBaoConnection` validates an OpenBao address and token Secret, then records health and authentication status.
- `OpenBaoPolicy` reconciles a named OpenBao ACL policy document, including explicit creation/adoption, drift correction, and optional deletion.
- `OpenBaoEntity` creates, adopts, updates, observes, and optionally deletes one OpenBao identity entity. The Kubernetes object name is the OpenBao entity name.
- `OpenBaoEntityAlias` binds an auth-method mount accessor and alias name to a referenced entity, with explicit adoption and deletion policies.
- `OpenBaoGroup` creates, adopts, updates, observes, and optionally deletes an OpenBao identity group.
- `OpenBaoGroupMembership` claims one entity or subgroup relationship for an internal group. The group controller owns only claimed edges and preserves other remote memberships.
- `internal/controller/openbao` contains reconciliation and dependency handling.
- `internal/openbaoclient` contains the intentionally small typed HTTP client.
- `config/` contains Kustomize installation and generated CRD/RBAC output.
- `hack/openbao-openapi.json` is a checked-in reference for the newest stable OpenBao release selected by `docs/compatibility.md`; it is not a generator input.
- `hack/e2e-kind.sh` and `hack/kind-*.yaml` define the disposable live integration environment.

## Sources of truth and boundaries

- Edit API types, controller/client source, samples, scripts, and documentation directly.
- Do not edit `api/**/zz_generated.deepcopy.go`, `config/crd/bases/`, or `config/rbac/role.yaml`; regenerate them with `make manifests generate`.
- `PROJECT` is Kubebuilder metadata. Change it only when the project layout or API inventory changes.
- OpenBao credentials belong only in same-namespace Kubernetes Secrets. Never put tokens in status, logs, samples, fixtures, or documentation.
- Connection and external identity references are same-namespace and immutable for `OpenBaoEntity`, `OpenBaoEntityAlias`, and `OpenBaoGroup`; changing the target requires deleting and recreating the resource. Group membership references are also immutable and require exactly one entity or subgroup target.
- `OpenBaoEntity` defaults to `creationPolicy: Create` and `deletionPolicy: Orphan`. External deletion is always opt-in.
- `OpenBaoGroup` defaults to `creationPolicy: Create` and `deletionPolicy: Orphan`; `OpenBaoGroupMembership` never deletes a group or entity.
- Preserve unrelated work in a dirty worktree. Generated files are derived output and should be reviewed for drift, not hand-edited.

## Canonical commands

```sh
make check             # Generate, format-check, vet, test, lint, validate OpenAPI, render Kustomize
make test              # Focused local unit tests
make manifests generate
make verify-generated  # Regenerate and compare tracked generated output
make build             # Build bin/manager
make build-installer   # Write dist/install.yaml
make run               # Run against the current kubeconfig context
make openapi-check     # Validate the checked-in OpenBao API reference
make docs-build        # Generate the CRD reference and build the strict docs site
make docs-serve        # Serve the docs site locally on localhost:8000
make kind-e2e          # Run live OpenBao/Kubernetes scenarios in Kind
make kind-e2e-clean    # Remove retained live E2E resources in dependency order
make kind-down         # Delete only the named local Kind cluster
make update-openbao-openapi OPENBAO_TOKEN=...  # Refresh from a running OpenBao instance
```

`make update-openbao-openapi` requires a running OpenBao instance and uses its `/v1/sys/internal/specs/openapi` endpoint. Set `OPENBAO_ADDR` for a non-default address. The generated document can vary with the OpenBao version and enabled mounts.

## Durable project knowledge

- `docs/index.md` is the documentation map.
- `docs/product.md` defines the current product promise and non-goals.
- `docs/user-stories.md` records current and future stories, acceptance criteria, and design coverage.
- `docs/architecture.md` describes component boundaries and reconciliation flow.
- `docs/research/2026-09-16-openbao-api.md` records the OpenBao API evidence used for this foundation.
- `docs/compatibility.md` defines the newest-stable-only OpenBao support policy and update procedure.
- `docs/decisions/` contains accepted consequential design decisions.
- `docs/backlog.md` contains real planned outcomes not implemented yet.
- `docs/verification.md` explains what checks prove and what they do not prove.
- `docs/operations/local-kind.md` documents the disposable Kind/OpenBao environment.
- `docs/crds/` contains behavior-oriented guides for each public custom resource.
- `docs/tech-debt.md` records material current limitations and their exit criteria.
- `docs/decisions/0005-test-pyramid.md` defines the boundary between unit, HTTP contract, and live Kind tests.
- `docs/reference/api.md` is generated by `make generate-api-reference`; do not edit it directly.
- `zensical.toml` defines the strict documentation-site navigation and validation.

Keep categories separate: put product promises in the product brief, external evidence in research, accepted rationale in decisions, planned outcomes in the backlog, unresolved current shortcomings in a tech-debt register, and implementation detail in code and tests. Update or remove stale guidance rather than appending exceptions.
