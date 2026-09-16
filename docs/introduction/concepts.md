# Concepts

## Connections

Every OpenBao-backed resource references an `OpenBaoConnection` in the same
Kubernetes namespace. A connection contains:

- the OpenBao API address;
- an optional OpenBao namespace;
- exactly one authentication method: a token Secret, Kubernetes Auth, or AppRole;
- an optional CA bundle Secret; and
- a per-request timeout.

Connection references are immutable. Create a new connection when a resource
must move to another OpenBao instance or namespace.

## Desired state

Kubernetes custom resources are the desired state and OpenBao is the external
system being reconciled:

- edits in Kubernetes drive OpenBao changes;
- periodic reconciliation corrects OpenBao-side drift;
- a connection becoming ready or changing credentials requeues dependents;
- the Kubernetes `metadata.name` is the external name for policies, entities,
  and groups; and
- relationships use Kubernetes references and observed status instead of
  hand-written OpenBao IDs.

## Resource relationships

```text
OpenBaoConnection
       │
       ├── OpenBaoPolicy
       ├── OpenBaoEntity ── OpenBaoEntityAlias
       └── OpenBaoGroup ── OpenBaoGroupMembership ──┐
                                                    └── entity or subgroup
```

An alias references an entity and binds an auth-method mount accessor and alias
name to that entity. A membership claims one entity or subgroup edge for an
internal group. Unclaimed remote group members remain untouched.

## Creation and adoption

Named resources use an explicit `creationPolicy`:

- `Create` creates a missing object and reports a conflict for an existing
  object that has not already been acquired;
- `Adopt` manages an existing object but does not create a missing object; and
- `CreateOrAdopt` creates when missing and adopts when present.

This prevents a new Kubernetes resource from silently overwriting an unrelated
OpenBao object. Use adoption only when the existing object is intentionally
being brought under management.

## Drift and status

Controllers read the external object periodically and update it when it differs
from the desired spec. Resources expose a `Ready` condition and
`observedGeneration`. Entities and aliases record stable OpenBao IDs; policies
record the OpenBao policy version and SHA-256 hash; groups and memberships
record the IDs needed to maintain their relationships.

## Deletion

`deletionPolicy` defaults to `Orphan` for named resources. The operator removes
the Kubernetes finalizer without deleting the external object unless
`deletionPolicy: Delete` is explicitly selected. Group memberships are claims:
deleting one removes only its claimed relationship, never the group or entity.

See [Deletion and ownership](../reference/deletion-and-ownership.md) for the
failure and dependency-loss behavior.
