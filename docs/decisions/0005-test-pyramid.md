# 0005 — Keep live E2E coverage narrow

## Status

Accepted

## Context

The operator has several reconcilers with create, adopt, drift, deletion,
dependency, and failure branches. Repeating those branches in a large raw
shell script makes failures slow to diagnose and causes the live suite to
become a second implementation of controller behavior. The difficult parts to
exercise live are the Kubernetes installation boundary, projected Kubernetes
Auth, and a small amount of real OpenBao wiring.

## Decision

Use three complementary verification layers:

1. Unit and fake-client reconciliation tests cover controller state transitions,
   ownership, status, finalizers, dependency watches, drift, and failure
   handling.
2. `httptest` client contract tests cover OpenBao paths, headers, request and
   response shapes, authentication, renewal, and retry behavior.
3. Kind E2E covers chart installation, CRD/RBAC presence, a real Kubernetes
   Auth login, and one successful resource graph using policy, entity, alias,
   group, and membership resources. Keep only scenarios that require a real
   Kubernetes/OpenBao boundary or are materially difficult to reproduce in
   unit tests.

The live suite should not duplicate every creation-policy, deletion-policy,
drift, conflict, or dependency-loss branch. New behavior belongs in unit and
HTTP contract tests first; add a live scenario only when it proves wiring that
those tests cannot observe.

## Consequences

The E2E workflow stays short enough to run routinely and its failures point to
installation, authentication, or integration wiring. Unit and contract suites
must remain comprehensive as controllers evolve, and the verification guide
must keep the boundary between test layers explicit.

This decision does not prohibit adding one focused live regression for an
OpenBao behavior that differs from its documented HTTP contract.
