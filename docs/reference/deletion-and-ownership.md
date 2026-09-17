# Deletion and ownership

The operator treats external OpenBao objects as shared infrastructure unless a
resource explicitly claims ownership. This makes the safe path the default for
new Kubernetes resources.

## Creation policy

Named resources support:

| Policy | Missing external object | Existing unacquired object |
| --- | --- | --- |
| `Create` | Create it | Report a conflict |
| `Adopt` | Report that it is missing | Adopt and reconcile it |
| `CreateOrAdopt` | Create it | Adopt and reconcile it |

Use `Adopt` when importing an object that is already managed outside
Kubernetes. The operator records stable IDs or an acquisition marker before it
starts treating later observations as its own managed object.

## Deletion policy

| Policy | Kubernetes deletion | OpenBao deletion |
| --- | --- | --- |
| `Orphan` | Remove the Kubernetes resource | Leave the external object |
| `Delete` | Wait for cleanup through a finalizer | Delete the external object |

`Orphan` is the default for policies, entities, aliases, and groups. A
membership resource is different: it always manages only its claimed edge and
never deletes its group or entity.

## Finalizers and dependency loss

Resources with `deletionPolicy: Delete` receive a finalizer before external
mutation. The finalizer is removed only after OpenBao confirms deletion or the
object is already absent.

If the connection or credential Secret disappears first, the controller
retains the finalizer, records `CleanupRequired=True` and `Stalled=True`, and
retries. Restore the same connection or credential so the operator can prove
that the external object was deleted. This intentionally keeps the Kubernetes
object `Terminating` rather than silently orphaning the external object.

Inspect a blocked resource before taking manual action:

```sh
kubectl get openbaoentity/example -o yaml
kubectl describe openbaoentity/example
kubectl get events --sort-by=.lastTimestamp
```

Do not remove a finalizer merely to hide a connectivity problem when external
cleanup is required. Manual finalizer removal is an administrative override
that can orphan the external object.
