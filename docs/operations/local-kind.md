# Local Kind environment

The local environment is disposable and isolated. The default workflow uses Kind's built-in CNI, a pinned OpenBao dev image, and the operator built from this checkout. Cilium is available for scenarios that need CNI-backed network-policy behavior.

## Start and test

```sh
make kind-e2e
```

The target:

- creates the named `openbao-entity-operator` Kind cluster;
- uses Kind's default CNI unless `KIND_CNI=cilium` is supplied;
- loads `openbao/openbao:2.6.2` and starts a single in-memory dev server;
- generates a local-only root token into a Kubernetes Secret, without writing it to the checkout or printing it;
- builds and loads the operator image;
- installs the generated CRDs and operator manifests;
- verifies connection health and token authentication;
- verifies entity creation, status ID persistence, metadata/policy/disabled-state updates, external deletion recovery, adoption, conflict protection, orphaning, and opt-in deletion;
- verifies alias creation, canonical-entity drift correction, adoption, conflict protection, orphaning, and opt-in deletion;
- verifies group creation, entity and subgroup membership, preservation of an unmanaged remote member, membership-claim removal, and opt-in group deletion;
- removes the test namespace after a successful run.

The default CNI is the recommended first run because the scenarios test reconciliation and API behavior. It does not prove NetworkPolicy enforcement. To use Cilium, create the cluster with:

```sh
make kind-e2e KIND_CNI=cilium
```

The default and Cilium modes use the same named cluster. Run `make kind-down` before switching modes.

## Inspect a failed run

Failed runs preserve the test namespace so status and logs remain available:

```sh
kubectl --context kind-openbao-entity-operator get pods -A
kubectl --context kind-openbao-entity-operator get openbaoconnections,openbaoentities,openbaogroups,openbaogroupmemberships -n openbao-entity-operator-e2e
kubectl --context kind-openbao-entity-operator describe openbaoentity/e2e-created -n openbao-entity-operator-e2e
kubectl --context kind-openbao-entity-operator logs deployment/openbao-entity-operator-controller-manager -n openbao-entity-operator-system
```

Set `KEEP_TEST_RESOURCES=true` to retain the namespace after a successful run too. Do not print or copy the `openbao-dev-token` Secret.

## Cleanup

Delete only the named disposable cluster:

```sh
make kind-down
```

This removes the local OpenBao data, token Secret, test resources, and operator deployment. It does not touch another Kubernetes context or cluster.

## Variables

Useful overrides include `KIND_CLUSTER`, `KIND_NODE_IMAGE`, `OPENBAO_IMAGE`, `OPENBAO_NAMESPACE`, `OPENBAO_TOKEN_SECRET`, and `CILIUM_VERSION`. The Cilium mode requires Helm and uses the pinned Cilium chart version from the root Makefile.
