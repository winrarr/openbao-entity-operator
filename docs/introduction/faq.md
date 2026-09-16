# FAQ

## Does this operator install OpenBao?

No. It manages resources in an existing OpenBao instance. OpenBao deployment,
storage, auth-method configuration, TokenReview credentials, and secret-engine
configuration are outside the current scope.

## Is this a Vault operator?

No compatibility promise is made for Vault distributions. The API and client
target OpenBao endpoints and behavior.

## Where are OpenBao tokens stored?

Input tokens are read from same-namespace Kubernetes Secrets and retained only
in memory by the controller client. Kubernetes Auth JWTs are read from the
projected ServiceAccount token path. Credentials are not written to resource
status, logs, or generated documentation.

## Can I manage an existing policy, entity, or group?

Yes. Set `creationPolicy: Adopt` when the object already exists and should be
managed. Use `CreateOrAdopt` when either creation or adoption is acceptable.
The controller refuses to overwrite an unacquired existing object under the
default `Create` policy.

## What happens if someone changes OpenBao directly?

The next reconciliation observes the difference and restores the Kubernetes
specification. Configure `spec.driftDetectionInterval` when a shorter or longer
periodic check is needed.

## Can resources reference a connection in another namespace?

No. Connections, Secrets, policies, entities, aliases, groups, and membership
references are namespace-bound. This keeps credential and tenancy boundaries
explicit.

## Does deleting a Kubernetes resource delete OpenBao data?

Not by default. Named resources default to `deletionPolicy: Orphan`; select
`Delete` explicitly when the external object should be removed too.
