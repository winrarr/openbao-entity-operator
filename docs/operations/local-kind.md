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
- removes the temporary test namespaces after a successful run.

After a successful run, the target removes the temporary test namespaces and
restores the operator's default cluster-wide Helm installation. A failed run
keeps the scoped installation and both test namespaces for inspection.

Each run first removes only the workflow's fixed `e2e-*` OpenBao fixtures from the disposable OpenBao instance. Do not point this workflow at a shared OpenBao deployment.

The root token is used only to bootstrap the disposable server and configure the
test auth methods. The main connection and resource graph use Kubernetes Auth
through the projected operator ServiceAccount JWT, while a separate connection
proves AppRole credential loading. Token-authenticated connection behavior is
covered by HTTP and reconciliation tests rather than by duplicated live
fixtures.

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

The cleanup target deletes membership claims, aliases, policies, groups, entities, and connections in that order, then removes the test namespaces. It stops if a resource remains blocked by a non-recoverable finalizer so the failure is visible. It does not delete external OpenBao objects that were left behind after a missing connection or credential.

Set `KEEP_TEST_RESOURCES=true` to retain the namespace after a successful run too. Do not print or copy the `openbao-dev-token` Secret.

## Cleanup

Delete only the named disposable cluster:

```sh
make kind-down
```

This removes the local OpenBao data, token Secret, test resources, and operator deployment. It does not touch another Kubernetes context or cluster.

## Variables

Useful overrides include `KIND_CLUSTER`, `KIND_NODE_IMAGE`, `E2E_TEST_NAMESPACE`, `E2E_OUTSIDE_NAMESPACE`, `OPENBAO_IMAGE`, `OPENBAO_NAMESPACE`, `OPENBAO_TOKEN_SECRET`, and `CILIUM_VERSION`. The Cilium mode requires Helm and uses the pinned Cilium chart version from the root Makefile.
