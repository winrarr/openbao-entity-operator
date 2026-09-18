# 0001 — Use a narrow typed HTTP client

- Status: Accepted
- Date: 2026-09-16

## Decision

Use a small typed HTTP client in `internal/openbaoclient` for each supported
OpenBao API family. The client remains resource-specific rather than exposing
an arbitrary path reconciler.

## Rationale

OpenBao's OpenAPI document is generated dynamically and is not a stable
code-generation input. A typed client makes request headers, status handling,
response validation, error redaction boundaries, credential exclusions, and
test doubles explicit while still allowing the operator to cover durable
configuration broadly.

## Alternatives considered

- Import the full OpenBao API/SDK package. This would provide broader coverage but brings a larger dependency surface and still does not turn the runtime OpenAPI document into a stable contract.
- Build a generic arbitrary-path reconciler. This would reduce initial client code but make CRD semantics, safety, and status behavior less explicit and harder to review.

The client boundary remains open for adding typed OpenBao-native operations
when a story has concrete acceptance criteria. It excludes arbitrary secret
data, credentials, diagnostics, and one-shot administrative actions whose
semantics are not durable reconciliation.
