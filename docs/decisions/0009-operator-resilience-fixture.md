# Decision 0009: Use a disposable persistent TLS fixture for operator resilience

Status: accepted

## Context

The fast Kind workflow uses OpenBao dev mode. That is useful for ordinary live reconciliation, but it cannot exercise the operator's behavior across a real server process restart, a persistent data path, TLS trust configuration, or a rotated static token. Reproducing those boundaries in unit tests would test mocks and timing rather than the operator's real Kubernetes and HTTP integration.

The project is not an OpenBao test suite. Adding a large end-to-end shell harness that asserts OpenBao storage, Raft, ACL, TLS, or high-availability behavior would duplicate OpenBao's own coverage and make the operator's verification surface slow and brittle.

## Decision

Keep the default live workflow fast and add an opt-in `make kind-e2e-resilience` profile. The profile provisions a disposable one-node OpenBao instance with a PVC, Raft storage, and a locally generated TLS certificate. Bootstrap and cleanup are test-fixture plumbing. The operator receives a non-root token through a Kubernetes Secret and a CA through a separate Secret.

The profile asserts only operator outcomes:

1. an `OpenBaoConnection` becomes Ready over TLS;
2. an `OpenBaoPolicy` reconciles through that connection;
3. a new policy reconciles after the OpenBao Pod is restarted;
4. revoking token A causes connection failure, rotating the Kubernetes Secret to token B restores readiness, and a new policy reconciles.

The root token is used only by the fixture bootstrap and remote-oracle cleanup commands. Assertions that initialize, unseal, or query the fixture are guards that make the operator scenario runnable; they are not product claims about OpenBao.

## Consequences

- The operator has a narrow live check for TLS trust, process restart recovery, and static credential rotation.
- Unit tests remain responsible for controller state transitions and the HTTP client contract.
- The profile is slower and requires local `openssl`, `jq`, Kind, Kubernetes, Helm, and a container runtime.
- A failed run retains its two fixed namespaces for inspection and has a dependency-aware cleanup target.
- The profile does not establish claims about OpenBao persistence, Raft correctness, ACL semantics, TLS implementation, HA, backup, or restore.
