# Initial user stories and design coverage

## Context

The primary actor is a platform operator who manages OpenBao configuration through Kubernetes. The current slice covers one connection, entity and group lifecycle, entity-alias binding, and explicit group membership claims.

## Current stories

### US-001 — Validate an OpenBao connection

As a platform operator, I want to declare an OpenBao address and token Secret, so that dependent resources can know whether OpenBao is reachable and the token is accepted.

Acceptance criteria:

- Given a reachable initialized OpenBao instance and valid token Secret, when the connection reconciles, then `Ready=True`, `Authenticated=True`, and the observed server version is recorded.
- Given a missing Secret, invalid address, or empty token key, when the connection reconciles, then `Ready=False` with a useful reason and no credential value in status or logs.
- Given a sealed, standby, or uninitialized instance, when health is checked, then the health state is represented in status and the connection does not claim successful authentication unless token self-lookup succeeds.

Design criteria: same-namespace Secret reference, bounded HTTP requests, OpenBao health status decoding, token redaction, and a watch/retry path for dependency recovery.

### US-002 — Manage an OpenBao entity declaratively

As a platform operator, I want a Kubernetes resource to create and continuously reconcile an OpenBao entity, so that identity metadata, policies, and disabled state remain aligned with declared intent.

Acceptance criteria:

- Given a ready connection and an absent entity, when an `OpenBaoEntity` is reconciled with `creationPolicy=Create`, then OpenBao contains an entity named after the Kubernetes resource and the returned ID is stored in status.
- Given a matching entity already exists, when `creationPolicy=Create`, then reconciliation reports a conflict and does not silently adopt it.
- Given a matching entity already exists, when `creationPolicy=Adopt` or `CreateOrAdopt`, then reconciliation records its ID and manages it.
- Given an entity is bound by ID, when metadata, policies, disabled state, or externally changed state differs, then reconciliation converges OpenBao and status to the declared state.

Design criteria: stable external ID binding, name mismatch protection, deterministic set comparison for policies, explicit create/adopt behavior, and injectable client contracts.

### US-003 — Apply safe deletion policy

As a platform operator, I want deletion behavior to be explicit, so that removing a Kubernetes object does not unexpectedly destroy an OpenBao identity.

Acceptance criteria:

- Given `deletionPolicy=Orphan`, when the Kubernetes resource is deleted, then the operator removes its finalizer without deleting the OpenBao entity.
- Given `deletionPolicy=Delete`, when the Kubernetes resource is deleted, then the operator deletes the bound OpenBao entity before removing its finalizer.
- Given OpenBao already reports the entity absent, when deletion is reconciled, then the finalizer is removed successfully.
- Given the referenced connection or credential Secret is already absent, when a Delete-policy resource is deleted, then the operator logs the dependency loss and removes its finalizer without claiming that the external object was deleted.

Design criteria: finalizer only for external deletion, idempotent not-found handling, dependency-loss escape hatch with an explicit warning, and no remote mutation before the delete policy is known.

### US-004 — Surface dependencies and failures

As a platform operator, I want dependency and API failures in Kubernetes status, so that `kubectl` is enough to diagnose why an entity is not ready.

Acceptance criteria:

- Given the referenced connection is missing or not ready, when the entity reconciles, then it reports `Ready=False` and does not call OpenBao.
- Given OpenBao returns an authentication or API error, when reconciliation fails, then the entity remains retryable and its condition includes the relevant reason without including the token.
- Given the external entity disappears, when periodic drift detection runs, then the operator reacquires it according to the declared creation policy.

Design criteria: condition reason stability, retry intervals, dependency watches, and status as the diagnostic contract.

### US-005 — Manage entity aliases

As a platform operator, I want to bind an OpenBao auth-method alias to an entity, so that authenticated workloads resolve to the declaratively managed identity.

Acceptance criteria:

- Given a ready connection and entity, when an `OpenBaoEntityAlias` is reconciled with `creationPolicy=Create`, then OpenBao contains the declared alias and the returned alias ID and canonical entity ID are stored in status.
- Given a matching alias already exists, when `creationPolicy=Create`, then reconciliation reports a conflict and does not silently adopt it.
- Given a matching alias already exists, when `creationPolicy=Adopt` or `CreateOrAdopt`, then reconciliation records its ID and manages it.
- Given an alias is bound by ID, when its canonical entity ID differs from the referenced entity's current ID, then reconciliation updates the alias to the referenced entity.
- Given `deletionPolicy=Orphan`, when the Kubernetes resource is deleted, then the OpenBao alias remains; given `deletionPolicy=Delete`, then the alias is removed before the finalizer is released.

Design criteria: same-namespace connection and entity references, immutable alias identity fields, stable alias IDs, alias-list lookup for initial adoption, explicit ownership policies, and watches for connection and entity changes.

### US-006 — Manage groups and membership

As a platform operator, I want to manage OpenBao identity groups and their entity membership, so that shared policies can be assigned to teams without duplicating policy configuration on every entity.

Acceptance criteria:

- Given a ready connection and an absent internal group, when an `OpenBaoGroup` is reconciled with `creationPolicy=Create`, then OpenBao contains a group named after the Kubernetes resource and the returned ID is stored in status.
- Given a matching group already exists, when `creationPolicy=Create`, then reconciliation reports a conflict; `Adopt` and `CreateOrAdopt` explicitly allow management of it.
- Given `deletionPolicy=Orphan`, when an `OpenBaoGroup` is deleted, then only the Kubernetes resource is removed; given `deletionPolicy=Delete`, then the external group is deleted before its finalizer is released.
- Given an `OpenBaoGroupMembership` references a ready entity or subgroup, when the parent group reconciles, then the corresponding OpenBao membership is present and the membership status records the parent and member IDs.
- Given a membership claim is deleted, when the parent group reconciles, then only that claimed relationship is removed; other remote memberships and the parent group remain.
- Given a remote membership not claimed by Kubernetes exists, when a group reconciles, then that membership remains untouched.
- Given an external group has a membership claim, when it reconciles, then the claim is rejected with a useful status condition because OpenBao manages external-group membership through an external alias.

Design criteria: same-namespace immutable references, stable group IDs, explicit create/adopt and deletion policies, internal-group membership claims, deterministic owned-edge tracking, preservation of unclaimed remote memberships, and watches for all referenced resources.

## Future stories

### US-007 — Support OpenBao namespaces when needed

As a platform operator, I want a connection or resource to target an OpenBao namespace, so that one operator can manage isolated identity domains in an OpenBao deployment that uses namespaces.

Reason to preserve: namespace context changes the effective API target and credential boundary. The current connection/client boundary should keep request context centralized without adding namespace behavior before it is needed.

## Design alternatives and recommendation

| Story | Status | Narrow typed HTTP client + explicit CRDs | Full OpenBao SDK + generic resource layer | Notes |
| --- | --- | --- | --- | --- |
| US-001 | Current | Covered now | Covered now | Typed health/auth contract is smaller and easier to test |
| US-002 | Current | Covered now | Covered now | Explicit create/adopt and stable ID are important in either design |
| US-003 | Current | Covered now | Supported later | Generic layers tend to obscure deletion safety |
| US-004 | Current | Covered now | Supported later | Status and dependency behavior belongs in controllers, not SDK calls |
| US-005 | Current | Covered now | Covered now | Alias client/controller binds to entity status ID without changing entity identity |
| US-006 | Current | Covered now | Covered now | Group membership is an explicit owned edge while the typed client models OpenBao's group-level update API |
| US-007 | Future | Supported later | Supported later | Centralized client construction preserves the extension point |

Recommend the narrow typed HTTP client with explicit CRDs. It covers the current stories with a small reviewable surface, keeps token handling and deletion semantics visible, and supports future OpenBao-native resources incrementally. The deliberate limitation is that each future endpoint needs a typed contract and focused tests; that cost is preferable to an arbitrary-path API whose safety is difficult to prove.

Verification for future extensibility: new resources should use stable OpenBao IDs, explicit same-namespace references, fake-client injection, OpenBao HTTP contract tests, and a live integration test before the feature is considered complete.
