# Backlog

These are real outcomes that are intentionally not part of the current supported slice.

## BL-001: Add additional OpenBao machine-auth providers

Goal: support a deliberately selected machine-auth provider such as AppRole for
deployments where Kubernetes Auth is unavailable.

Rationale: token Secrets and Kubernetes Auth cover the current deployment
boundaries, but some installations already have another OpenBao credential
provisioning system. Supporting it would reduce the need for long-lived static
tokens without turning the connection API into an arbitrary login-payload
surface.

Constraints: each provider must have an explicit typed CRD shape, a documented
credential rotation lifecycle, redaction guarantees, focused HTTP contract
tests, and a live test fixture. Provider configuration remains outside this
operator's responsibility unless a separate story explicitly says otherwise.

Acceptance criteria:

- A supported provider can be selected without changing identity, alias, group,
  or membership controllers.
- Credentials are read only from an explicitly scoped source, are never written
  to status or logs, and rotate without requiring an operator restart.
- Authentication failures and provider-specific lease behavior are represented
  through the existing connection conditions.
- The unit suite covers login, renewal or rotation, and unauthorized retry
  behavior; the isolated Kind or integration suite covers a real provider
  lifecycle.
