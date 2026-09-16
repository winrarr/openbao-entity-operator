# 0003 — Scope namespace context to OpenBaoConnection

- Status: Accepted
- Date: 2026-09-16

## Decision

Add an optional, immutable `namespace` path to `OpenBaoConnection`. The typed OpenBao client sends it as `X-Vault-Namespace` on every request; an empty value preserves root-namespace behavior.

## Rationale

OpenBao namespaces change the effective identity, policy, and authentication context of API requests. Keeping the value on the connection makes the boundary explicit and ensures authentication, entities, aliases, groups, and memberships use the same context. Root connections use `/sys/health` for server observations; OpenBao rejects that endpoint from within a namespace, so namespace-scoped connections use token self-lookup to establish readiness. Immutability prevents an existing external ID from silently being interpreted in a different namespace after a connection update.

The API validates path shape and the client rejects OpenBao-reserved or otherwise invalid path segments before making a request. Namespace lifecycle management is outside this operator's scope; the targeted namespace must already exist and the configured authentication method must be authorized within it.

## Consequences

- All resources using one connection share its namespace context.
- Empty namespace values remain compatible with non-namespaced OpenBao deployments.
- Moving resources between namespaces requires a new connection and explicit resource recreation or adoption.
- Namespace deletion and credential loss can still leave external resources unavailable for cleanup; existing deletion-policy behavior applies.
