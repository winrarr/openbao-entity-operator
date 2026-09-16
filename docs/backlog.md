# Backlog

These are real outcomes that are intentionally not part of the first vertical slice.

## OpenBao namespaces

Goal: add OpenBao namespace targeting when the supported deployment model requires isolated identity domains and credential boundaries.

Rationale: namespace context changes the effective API target and token scope, so it should be designed at the connection/client boundary rather than added ad hoc to individual resources.

Constraints: preserve same-namespace Kubernetes references, keep namespace headers and URL handling inside the typed client, and do not claim namespace support until the live workflow covers it.

Acceptance criteria:

- A connection can target an OpenBao namespace without duplicating namespace request logic across controllers.
- Health, authentication, entity, alias, group, and membership requests use the selected namespace consistently.
- Missing or invalid namespace configuration is visible through connection and dependent-resource status.
- A live test proves that two OpenBao namespaces remain isolated.

## Installation chart

Goal: add a Helm chart when the supported installation surface needs chart values or release packaging beyond the generated Kustomize bundle.

Rationale: the generated Kustomize bundle is sufficient for the current development and installation workflow; a chart should be justified by a real packaging or configuration need.

Constraints: preserve the generated CRD and RBAC semantics, document ownership between chart templates and generated manifests, and keep image, namespace, metrics, and security settings explicit.

Acceptance criteria:

- The chart installs the same CRDs, controller, RBAC, and metrics resources as the supported Kustomize bundle.
- Image, namespace, metrics, and security settings are configurable and validated.
- A chart render and install test runs in the supported Kubernetes test environment.
