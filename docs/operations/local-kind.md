# Local Kind environment

The local environment is disposable and isolated. The default workflow uses Kind's built-in CNI, a pinned OpenBao dev image, and the operator built from this checkout. Cilium is available for scenarios that need CNI-backed network-policy behavior.

## Start and test

```sh
make kind-e2e
```

The target builds the operator image, installs the committed Helm chart, and then:

- creates the named `openbao-entity-operator` Kind cluster;
- uses Kind's default CNI unless `KIND_CNI=cilium` is supplied;
- loads `openbao/openbao:2.6.2` and starts a single in-memory dev server;
- generates a local-only root token into a Kubernetes Secret, without writing it to the checkout or printing it;
- configures OpenBao Kubernetes Auth for the operator ServiceAccount and gives that role only the identity, ACL policy, and token lifecycle permissions needed by the test;
- builds and loads the operator image;
- installs the generated CRDs and operator Helm chart;
- verifies CRD installation, namespace-scoped manager RoleBindings, and that a
  valid connection outside the configured watch namespace is not reconciled;
- configures OpenBao Kubernetes Auth and verifies a real projected ServiceAccount login;
- configures OpenBao AppRole and verifies a real role ID and Secret ID login;
- verifies one successful ACL policy and entity graph, including policy status version/hash and entity ID persistence;
- verifies one entity alias binding and one internal group membership against the live OpenBao API;
- verifies that a Delete-policy entity retains its finalizer while a token credential is unavailable, then completes external cleanup after the credential is restored;
- provisions two isolated OpenBao namespaces and platform-owned Kubernetes Auth
  connections;
- verifies tenant ServiceAccounts cannot read Secrets, manage connections, or
  access the other tenant's Kubernetes resources;
- verifies each tenant can reconcile its own policy/entity graph while a token
  from either OpenBao namespace cannot mutate the other namespace;
- removes the temporary test namespaces after a successful run.

After a successful run, the target removes the temporary OpenBao namespaces and
Kubernetes test resources, then removes the temporary test namespaces and
restores the operator's default cluster-wide Helm installation. A failed run
keeps the scoped installation and both test namespaces for inspection.

Each run first removes only the workflow's fixed `e2e-*` OpenBao fixtures from
the disposable OpenBao instance. The multi-tenancy scenario uses the fixed
`openbao-entity-operator-tenant-a` and
`openbao-entity-operator-tenant-b` namespace names and removes them after a
successful run. Do not point this workflow at a shared OpenBao deployment.

The root token is used only to bootstrap the disposable server and configure the
test auth methods. The main connection and resource graph use Kubernetes Auth
through the projected operator ServiceAccount JWT, while separate connections
prove AppRole credential loading and token-credential recovery during
Delete-policy cleanup.

The default CNI is the recommended first run because the scenarios test reconciliation and API behavior. It does not prove NetworkPolicy enforcement. To use Cilium, create the cluster with:

```sh
make kind-e2e KIND_CNI=cilium
```

The default and Cilium modes use the same named cluster. Run `make kind-down` before switching modes.

## Inspect a failed run

Failed runs preserve the test namespaces so status and logs remain available:

```sh
kubectl --context kind-openbao-entity-operator get pods -A
kubectl --context kind-openbao-entity-operator get openbaoconnections,openbaopolicies,openbaoentities,openbaoentityaliases,openbaogroups,openbaogroupmemberships -n openbao-entity-operator-e2e
kubectl --context kind-openbao-entity-operator describe openbaoentity/e2e-entity -n openbao-entity-operator-e2e
kubectl --context kind-openbao-entity-operator logs deployment/openbao-entity-operator -n openbao-entity-operator-system
```

After inspection, remove only the retained E2E resources with the dependency-aware cleanup target:

```sh
make kind-e2e-clean
```

The cleanup target deletes membership claims, aliases, policies, groups,
entities, and connections in that order, removes the two test namespaces, and
removes the fixed multi-tenancy OpenBao namespaces when the disposable server
is available. It stops if a resource remains blocked by a non-recoverable
finalizer so the failure is visible.

Set `KEEP_TEST_RESOURCES=true` to retain the namespace after a successful run too. Do not print or copy the `openbao-dev-token` Secret.

## Operator resilience profile

```sh
make kind-e2e-resilience
```

This opt-in profile keeps the default Kind/OpenBao setup as the platform environment, then installs the operator with a watch scope limited to a separate test namespace. It provisions a disposable one-node OpenBao fixture with a PVC, Raft storage, a locally generated CA, and a TLS listener. The fixture is bootstrapped out of band and gives the operator only a non-root token policy.

The assertions are intentionally operator-focused:

- a token-authenticated `OpenBaoConnection` becomes Ready over TLS with the referenced CA Secret;
- an `OpenBaoPolicy` reconciles through that connection;
- a new policy reconciles after the OpenBao Pod is restarted;
- revoking token A makes the connection lose readiness, replacing the Kubernetes Secret with token B restores readiness, and a new policy reconciles with token B.

The PVC, Raft initialization, unseal operation, TLS certificate generation, root bootstrap token, and policy creation are fixture plumbing. This profile does not test OpenBao storage correctness, Raft correctness, ACL semantics, HA/standby behavior, or backup and restore. Those are not operator claims.

The resilience fixture is retained when the target fails so logs and status can be inspected. Clean it and restore the normal cluster-wide Helm installation with:

```sh
make kind-e2e-resilience-clean
```

Do not point this workflow at a shared OpenBao deployment. The default values use `openbao-entity-operator-persistent` for the fixture namespace and `openbao-entity-operator-resilience` for the test namespace; both can be overridden when the names are available only in the disposable cluster.

## Cleanup

Delete only the named disposable cluster:

```sh
make kind-down
```

This removes the local OpenBao data, token Secret, test resources, and operator deployment. It does not touch another Kubernetes context or cluster.

## Variables

Useful overrides include `KIND_CLUSTER`, `KIND_NODE_IMAGE`,
`E2E_TEST_NAMESPACE`, `E2E_TENANT_B_NAMESPACE`, `E2E_OUTSIDE_NAMESPACE`,
`OPENBAO_IMAGE`, `OPENBAO_NAMESPACE`, `OPENBAO_TOKEN_SECRET`,
`PERSISTENT_OPENBAO_NAMESPACE`, `PERSISTENT_OPENBAO_DEPLOYMENT`,
`PERSISTENT_OPENBAO_SERVICE`, `PERSISTENT_OPENBAO_TLS_SECRET`,
`PERSISTENT_E2E_NAMESPACE`, and `CILIUM_VERSION`. The Cilium mode requires
Helm and uses the pinned Cilium chart version from the root Makefile.
