# 0004 — Make Helm the configurable installation surface

- Status: Accepted
- Date: 2026-09-16

## Decision

Support the operator Helm chart as the installation surface. Helm packages
CRDs from `config/crd/bases/` and wraps generated manager RBAC permissions with
chart-owned names, labels, and bindings.

## Rationale

The operator needs explicit configuration for image references, metrics,
security settings, and installation namespace. Helm provides those values and
release lifecycle semantics without changing the Kubernetes API or controller
behavior. Kustomize remains an internal Kubebuilder and sample-rendering tool,
not a supported installation interface.

## Consequences

- `make manifests` synchronizes generated CRDs and manager permissions into the chart.
- `make verify-generated`, `make helm-lint`, and `make helm-template` protect the chart in local and CI checks.
- `make deploy` and the live Kind workflow install the Helm release.
- The release workflow packages and publishes the chart with the Helm CLI.
