# Decision 0008: Retain deletion finalizers when cleanup access is unavailable

- Status: Accepted
- Date: 2026-09-17

## Decision

Delete-policy resources retain their finalizer when the referenced
`OpenBaoConnection`, credential Secret, or external deletion request is
unavailable. The controller records `CleanupRequired=True` and `Stalled=True`,
then retries until the cleanup dependency is restored or OpenBao confirms that
the external object is already absent.

## Rationale

Releasing a finalizer after losing the connection or credentials makes the
Kubernetes deletion complete without proving that the external OpenBao object
was deleted. Retaining the finalizer preserves the declared Delete policy and
keeps the cleanup operation recoverable: restoring the same connection or
credential allows reconciliation to continue without an administrative orphan
cleanup path.

The status condition makes the intentional `Terminating` state visible and
gives operators a direct recovery signal. The operator does not store or log
credential values.

## Consequences

- Platform operators must restore a referenced connection or credential before
  a Delete-policy resource can finish deletion.
- A lost connection or credential can intentionally leave a resource
  `Terminating`; removing its finalizer manually is an administrative choice
  that may orphan the external object.
- Orphan-policy resources are unaffected and still release their finalizer
  without requiring OpenBao access.
- Unit tests cover dependency loss and recovery, while the live Kind workflow
  covers a credential-loss deletion and recovery path.
