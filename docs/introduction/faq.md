# FAQ

## Does this operator install OpenBao?

No. It manages resources in an existing OpenBao instance. OpenBao deployment,
storage, HA, initialization, unseal, and server upgrades remain external. The
operator can configure durable auth-method and secret-engine records through
their CRDs after the server is available.

## Is this a Vault operator?

No compatibility promise is made for Vault distributions. The API and client
target OpenBao endpoints and behavior.

## Where are OpenBao tokens stored?

Input tokens and AppRole credentials are read from same-namespace Kubernetes
Secrets and retained only in memory by the controller client. Kubernetes Auth
JWTs are read from the projected ServiceAccount token path. Credentials are not
written to resource status, logs, or generated documentation.

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

## Does the operator manage AppRole configuration?

Yes, `OpenBaoAppRole` can manage the durable configuration of a role in an
already enabled AppRole mount. OpenBao administrators or another credential
process still own enabling the mount, issuing Role IDs and Secret IDs, and
delivering or rotating those credentials. `OpenBaoConnection` consumes role ID
and Secret ID values from same-namespace Secrets when it authenticates.

## Does the operator manage secret values or generated credentials?

No. It does not reconcile arbitrary secret-engine data, issue tokens or Secret
IDs, or copy generated AppRole and OIDC client secrets to Kubernetes.

## Does deleting a Kubernetes resource delete OpenBao data?

Not by default. Named resources default to `deletionPolicy: Orphan`; select
`Delete` explicitly when the external object should be removed too.
