# Troubleshooting

## Start with the resource status

Most reconciliation failures are recorded on the custom resource:

```sh
kubectl get <resource> <name> -o yaml
kubectl describe <resource> <name>
```

Read the `Ready` condition's `reason` and `message`, then compare
`observedGeneration` with `metadata.generation`.

## Connection failures

Common causes include:

- the OpenBao address is not reachable from the operator Pod;
- the referenced Secret is missing, in another namespace, or lacks its key;
- the token lacks the required capability;
- the Kubernetes Auth mount or role is absent or bound to the wrong
  ServiceAccount;
- the AppRole mount or role is absent, or the role ID or Secret ID Secret lacks
  its configured key;
- a projected ServiceAccount token is disabled; or
- the OpenBao CA is not trusted by the supplied CA bundle.

Inspect the connection and controller logs:

```sh
kubectl get openbaoconnection/<name> -o yaml
kubectl logs deployment/openbao-entity-operator \
  -n openbao-entity-operator-system
```

## Creation conflicts

`*AcquireFailed` or conflict conditions normally mean an external object
already exists while the resource still uses `creationPolicy: Create`. Confirm
the object is the intended one, then set `creationPolicy: Adopt` or
`CreateOrAdopt` explicitly.

## Policy drift or version changes

For policies, compare `spec.rules`, `status.rulesHash`, and the remote document.
An external edit is expected to be restored on the next reconciliation. If it
is not, check connection readiness, policy permissions, and controller logs.

## Group membership problems

Verify that the parent group and referenced entity or subgroup are both
`Ready=True`, that exactly one membership target is set, and that all
references are in the same namespace. The controller preserves remote members
that are not represented by a membership claim.

## Stuck finalizers

If a resource is `Terminating`, check its connection, deletion policy, and
events before changing anything:

```sh
kubectl get <resource> <name> -o jsonpath='{.metadata.finalizers}{"\n"}'
kubectl get events --sort-by=.lastTimestamp
```

`deletionPolicy: Delete` requires OpenBao access. If the connection or its
credentials disappeared first, the controller retains the finalizer and sets
`CleanupRequired=True` until the dependency is restored. Recreate the same
connection or credential Secret, then inspect the resource again. Remove a
finalizer manually only when intentionally accepting that the external object
may be orphaned.

## Local Kind failures

Use the [local Kind guide](../operations/local-kind.md) to inspect the retained
E2E namespaces and run `make kind-e2e-clean`. Use `make kind-down` only to
remove the named disposable cluster.
