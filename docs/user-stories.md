# Initial user stories and design coverage

## Context

The primary actor is a platform operator who manages OpenBao configuration through Kubernetes. The current slice covers token-, Kubernetes-authenticated, and AppRole connections, ACL policy lifecycle, Kubernetes Auth role lifecycle inside preconfigured mounts, entity and group lifecycle, entity-alias binding, and explicit group membership claims.

## Current stories

### US-001 — Validate an OpenBao connection

As a platform operator, I want to declare an OpenBao address and supported authentication method, so that dependent resources can know whether OpenBao is reachable and the operator is authorized to manage identities.

Acceptance criteria:

- Given a reachable initialized OpenBao instance and valid token Secret, when the connection reconciles, then `Ready=True`, `Authenticated=True`, and the observed server version is recorded.
- Given a missing Secret, invalid address, or empty token key, when the connection reconciles, then `Ready=False` with a useful reason and no credential value in status or logs.
- Given a sealed, standby, or uninitialized instance, when health is checked, then the health state is represented in status and the connection does not claim successful authentication unless token self-lookup succeeds.
- Given a configured Kubernetes auth mount and role, when the operator's projected ServiceAccount JWT is accepted, then the connection logs in and reports `Ready=True` without requiring a token Secret.
- Given a Kubernetes-authenticated token is renewable and approaching expiry, when a dependent request is made, then the token is renewed before the request proceeds.
- Given a Kubernetes-authenticated token is rejected as expired or revoked, when a request receives an authentication failure, then the operator obtains a fresh token and retries the request once.

Design criteria: exactly one authentication method per connection, same-namespace Secret references for static tokens and AppRole credentials, projected ServiceAccount JWTs for Kubernetes Auth, bounded HTTP requests, OpenBao health status decoding, credential redaction, renewable-token handling, and a watch/retry path for dependency recovery.

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
- Given the referenced connection or credential Secret is already absent, when a Delete-policy resource is deleted, then the operator retains its finalizer, reports `CleanupRequired=True` and `Stalled=True`, and retries until the dependency is restored.

Design criteria: finalizer only for external deletion, idempotent not-found handling, recoverable dependency-loss handling with explicit status, and no remote mutation before the delete policy is known.

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

### US-007 — Target isolated OpenBao namespaces

As a platform operator, I want a connection to target an OpenBao namespace, so that one operator can manage isolated identity domains in an OpenBao deployment that uses namespaces.

Acceptance criteria:

- Given a connection with no namespace, when it reconciles, then all requests target OpenBao's root namespace as before.
- Given a connection with a valid absolute or relative namespace path, when it reconciles, then namespaced token self-lookup succeeds and entity, alias, group, and membership requests use that namespace consistently; the connection is Ready without root-only health fields because OpenBao does not expose `sys/health` within a namespace.
- Given a missing or invalid namespace target, when the connection reconciles, then the connection reports `Ready=False` and dependent resources remain blocked without making external identity mutations.
- Given two connections target different OpenBao namespaces, when entities with the same desired shape are reconciled, then each entity is isolated to its selected namespace.

Design criteria: namespace context belongs to the connection/client boundary, root compatibility, immutable namespace targeting, OpenBao naming validation, and live isolation coverage.

### US-008 — Use Kubernetes Auth without static operator credentials

As a platform operator, I want the operator to authenticate with OpenBao using its Kubernetes ServiceAccount, so that the deployment does not require a long-lived OpenBao token Secret.

Acceptance criteria:

- Given OpenBao's Kubernetes auth method is configured for the operator ServiceAccount, when an `OpenBaoConnection` selects `kubernetesAuth`, then the operator logs in through the configured auth mount and dependent identity resources reconcile successfully.
- Given `kubernetesAuth.mountPath` is omitted, when the connection reconciles, then the default `kubernetes` auth mount is used.
- Given the auth role or projected ServiceAccount token is invalid, when the connection reconciles, then it reports `Ready=False` with a useful reason and does not attempt identity mutations.
- Given the Kubernetes auth token is renewable, when its lease approaches expiry, then the shared client renews it; if renewal fails or OpenBao rejects the token, then the next request performs one fresh login retry.
- Given a token-authenticated connection remains configured, when the new API is deployed, then its existing token Secret behavior remains unchanged.

Design criteria: authentication belongs to `OpenBaoConnection`, the client owns login/renew/retry behavior, JWT material is read only from the projected ServiceAccount file, auth mount paths are explicit and validated, and token or JWT values never enter status, errors, logs, or test fixtures.

### US-009 — Manage an OpenBao ACL policy declaratively

As a platform operator, I want to declare an OpenBao ACL policy document as a Kubernetes resource, so that authorization rules are versioned and continuously reconciled with the same connection and ownership controls as identity resources.

Acceptance criteria:

- Given a ready connection and an absent policy, when an `OpenBaoPolicy` is reconciled with `creationPolicy=Create`, then OpenBao contains a policy named after the Kubernetes resource and the resource reports `Ready=True` with its observed version and rules hash.
- Given an existing policy, when `creationPolicy=Create`, then reconciliation reports a conflict and does not overwrite the document; `Adopt` and `CreateOrAdopt` explicitly allow management.
- Given a managed policy whose OpenBao document changes or is deleted externally, when drift detection runs, then the declared `spec.rules` document is restored.
- Given `deletionPolicy=Orphan`, when the Kubernetes resource is deleted, then the OpenBao policy remains; given `deletionPolicy=Delete`, then the policy is removed before the finalizer is released.
- Given the referenced connection or credential Secret is already absent during Delete-policy cleanup, then the controller retains the finalizer, reports `CleanupRequired=True`, and resumes external deletion after the dependency is restored.

Design criteria: same-namespace immutable connection reference, Kubernetes resource name as the stable OpenBao policy name, exact raw HCL or JSON document comparison, status hash/version instead of duplicating the full policy, explicit create/adopt and deletion policy, typed ACL endpoints only, and live Kubernetes Auth coverage.

### US-010 — Authenticate with AppRole

As a platform operator, I want to authenticate the operator with OpenBao AppRole, so that I can use an existing machine-auth workflow when Kubernetes Auth is unavailable.

Acceptance criteria:

- Given an enabled AppRole mount and configured role, when an `OpenBaoConnection` selects `appRole` with valid role ID and Secret ID references, then the connection logs in through the configured mount and reports `Ready=True`.
- Given omitted AppRole mount path, when the connection reconciles, then the default `approle` mount is used.
- Given missing or empty role ID or Secret ID data, when the connection reconciles, then it reports `Ready=False` without exposing credential values in status, logs, or errors.
- Given a renewable AppRole token, when its lease approaches expiry, then the shared client renews it; if renewal fails or OpenBao rejects it, then both credential Secrets are reread and one fresh login is attempted.
- Given a token- or Kubernetes-authenticated connection remains configured, when AppRole support is deployed, then its existing behavior remains unchanged.

Design criteria: one typed AppRole configuration, separate same-namespace credential references, explicit mount validation, lazy credential reads, external Secret ID rotation ownership, reusable token lease handling, and no arbitrary authentication request payloads.

## Current stories

### US-011 — Operate with explicit Kubernetes tenant boundaries

As a platform operator, I want to scope an operator installation to an explicit tenant boundary, so that one tenant cannot use the operator as a confused deputy against another tenant's Kubernetes Secrets or OpenBao identity domain.

Acceptance criteria:

- Given an installation boundary, when resources are created outside its allowed Kubernetes namespaces, then they are not watched or reconciled.
- Given platform-owned tenant-scoped connections, credential boundaries, and an OpenBao identity domain, when a tenant resource reconciles, then it cannot select another tenant's Secret, connection, or domain.
- Given the supported tenant resource kinds and deletion policies, when admission and RBAC are configured, then the allowed operations and external blast radius are explicit and reviewable.
- Given a separately scoped installation and the tenant-author RBAC profile, when it is deployed, then its Kubernetes RBAC and OpenBao permissions are sufficient for each declared tenant boundary and insufficient for another tenant's boundary.

Design criteria: explicit watch and reference scope, least-privilege RBAC, platform-owned connection and credential material, stable external-domain binding, admission-policy integration, and focused boundary verification.

### US-012 — Manage Kubernetes Auth workload roles

As a platform operator, I want to declare the ServiceAccount bindings and token policy of an OpenBao Kubernetes Auth role, so that workload authentication is versioned and drift-corrected without putting auth-mount administration in the operator.

Acceptance criteria:

- Given a ready connection with permission to manage an existing Kubernetes Auth mount and an absent role, when an `OpenBaoKubernetesAuthRole` uses `creationPolicy=Create`, then the role is created at the selected `auth/<mount>/role/<name>` path and its normalized configuration hash is recorded in status.
- Given an existing role, when `creationPolicy=Create`, then reconciliation reports a conflict without overwriting it; `Adopt` and `CreateOrAdopt` explicitly allow management.
- Given a managed role whose ServiceAccount bindings, token policies, or token lifetimes change outside Kubernetes, when drift detection runs, then the declared role configuration is restored.
- Given `deletionPolicy=Orphan`, when the Kubernetes resource is deleted, then the OpenBao role remains; given `deletionPolicy=Delete`, then the role is deleted before its finalizer is released.
- Given the connection or its credentials are unavailable during Delete-policy cleanup, then the finalizer is retained, `CleanupRequired=True` is reported, and cleanup resumes after the dependency is restored.
- Given a tenant-author RBAC profile and a platform-owned connection whose OpenBao policy is scoped to the tenant domain, when a tenant manages a role in its namespace, then it cannot use the role resource to read or mutate another tenant's connection, credential, or OpenBao namespace.

Design criteria: preconfigured auth-mount boundary, immutable connection and mount references, explicit create/adopt and deletion policy, normalized set comparison, whole-second OpenBao duration encoding, native OpenBao ACL authorization, no credential status, and focused live tenant-boundary verification.

## Design alternatives and recommendation

| Story | Status | Narrow typed HTTP client + explicit CRDs | Full OpenBao SDK + generic resource layer | Notes |
| --- | --- | --- | --- | --- |
| US-001 | Current | Covered now | Covered now | Typed health/auth contract is smaller and easier to test |
| US-002 | Current | Covered now | Covered now | Explicit create/adopt and stable ID are important in either design |
| US-003 | Current | Covered now | Supported later | Generic layers tend to obscure deletion safety |
| US-004 | Current | Covered now | Supported later | Status and dependency behavior belongs in controllers, not SDK calls |
| US-005 | Current | Covered now | Covered now | Alias client/controller binds to entity status ID without changing entity identity |
| US-006 | Current | Covered now | Covered now | Group membership is an explicit owned edge while the typed client models OpenBao's group-level update API |
| US-007 | Current | Covered now | Supported later | The typed client applies one validated namespace header to every request |
| US-008 | Current | Covered now | Supported later | Authentication is selected at connection construction while identity controllers keep a narrow client interface |
| US-009 | Current | Covered now | Supported later | A typed policy client keeps the raw document and ownership semantics visible without exposing arbitrary system paths |
| US-010 | Current | Covered now | Covered now | AppRole extends the connection boundary with lazy credential sources and shared token lease handling |
| US-011 | Current slice | Covered now | Covered now | `watchNamespaces`, scoped Helm RoleBindings, the tenant-author RBAC profile, platform-owned connections, OpenBao namespaces, and the two-tenant Kind scenario cover the supported model |
| US-012 | Current | Covered now | Supported later | The role endpoint is a narrow typed extension; mount enablement and TokenReview configuration remain outside the operator, and OpenBao ACLs provide the runtime authorization boundary |

Recommend the narrow typed HTTP client with explicit CRDs. It covers the current stories with a small reviewable surface, keeps token handling and deletion semantics visible, and supports future OpenBao-native resources incrementally. The deliberate limitation is that each future endpoint needs a typed contract and focused tests; that cost is preferable to an arbitrary-path API whose safety is difficult to prove.

Verification for future extensibility: new resources should use stable OpenBao IDs, explicit same-namespace references, fake-client injection, OpenBao HTTP contract tests, and a live integration test before the feature is considered complete.
