# 0002 — Model group membership as explicit owned edges

- Status: Accepted
- Date: 2026-09-16

## Decision

Represent each desired OpenBao group membership as an `OpenBaoGroupMembership` resource. Let the `OpenBaoGroup` controller reconcile the group-level membership arrays using the union of current claims and retain remote member IDs that were never claimed by Kubernetes.

## Rationale

OpenBao's identity API updates `member_entity_ids` and `member_group_ids` on the group resource; it does not provide a standalone membership endpoint. A separate Kubernetes resource still gives each relationship an explicit, immutable reference and makes deleting a membership claim independent from deleting the parent group or member entity. Tracking managed IDs in group status prevents one claim from removing an unrelated remote membership.

External groups are intentionally rejected when a membership claim targets them. OpenBao documents external-group membership as being managed through an external group alias, so treating those member arrays as operator-owned would conflict with the external identity source.

## Consequences

- Membership claims are namespaced, same-namespace, and immutable after creation.
- Group status contains all observed member IDs plus the IDs currently owned through membership claims.
- Membership reconciliation is group-scoped and needs a read/update cycle for each changed group.
- A stale or unavailable dependency blocks membership convergence and is visible through both group and membership status.
