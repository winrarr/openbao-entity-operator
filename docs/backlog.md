# Backlog

These are real outcomes that are intentionally not part of the first vertical slice.

## Installation chart

Goal: add a Helm chart when the supported installation surface needs chart values or release packaging beyond the generated Kustomize bundle.

Rationale: the generated Kustomize bundle is sufficient for the current development and installation workflow; a chart should be justified by a real packaging or configuration need.

Constraints: preserve the generated CRD and RBAC semantics, document ownership between chart templates and generated manifests, and keep image, namespace, metrics, and security settings explicit.

Acceptance criteria:

- The chart installs the same CRDs, controller, RBAC, and metrics resources as the supported Kustomize bundle.
- Image, namespace, metrics, and security settings are configurable and validated.
- A chart render and install test runs in the supported Kubernetes test environment.
