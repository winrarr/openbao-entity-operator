# 0001 — Use a narrow typed HTTP client

- Status: Accepted
- Date: 2026-09-16

## Decision

Use a small typed HTTP client in `internal/openbaoclient` for the OpenBao connection and entity endpoints required by the first vertical slice.

## Rationale

The current product scope needs only health, token self-lookup, and identity entity operations. OpenBao's OpenAPI document is generated dynamically and is not a stable code-generation input. A narrow client makes request headers, status handling, response validation, error redaction boundaries, and test doubles explicit.

## Alternatives considered

- Import the full OpenBao API/SDK package. This would provide broader coverage but brings a larger dependency surface and still does not turn the runtime OpenAPI document into a stable contract.
- Build a generic arbitrary-path reconciler. This would reduce initial client code but make CRD semantics, safety, and status behavior less explicit and harder to review.

The client boundary remains open for adding typed OpenBao-native operations when a future story has concrete acceptance criteria.
