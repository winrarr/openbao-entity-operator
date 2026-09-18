# Product scope

## Promise

OpenBao Entity Operator lets a platform team declare OpenBao ACL policies, Kubernetes Auth roles, identity entities, and groups as Kubernetes resources. It manages them through an authenticated OpenBao connection and exposes readiness, external identity, membership, policy version/hash, role configuration hash, and reconciliation failures through Kubernetes status.

## Current scope

- Same-namespace `OpenBaoConnection` resources with an address, optional OpenBao namespace, exactly one of a token Secret, Kubernetes Auth using the operator ServiceAccount, or AppRole credentials, an optional CA bundle Secret, and request timeout.
- Same-namespace `OpenBaoPolicy` resources whose Kubernetes name is the OpenBao ACL policy name and whose `spec.rules` is the raw HCL or JSON policy document.
- Same-namespace `OpenBaoKubernetesAuthRole` resources whose Kubernetes name is the OpenBao Kubernetes Auth role name and whose mount is enabled and configured outside the operator.
- Same-namespace `OpenBaoEntity` resources whose Kubernetes name is the OpenBao entity name.
- Same-namespace `OpenBaoEntityAlias` resources that bind an auth-method alias name and mount accessor to an `OpenBaoEntity`.
- Same-namespace `OpenBaoGroup` resources for internal or external OpenBao identity groups.
- Same-namespace `OpenBaoGroupMembership` resources that claim one entity or subgroup relationship for an internal group.
- Create, adopt, update, observe, drift-correct, orphan, and opt-in delete behavior.
- Desired policy documents, entity/group metadata, ACL policy names, and disabled state.
- Status conditions that distinguish dependencies, configuration, authentication, and external API failures.
- Helm and generated Kustomize installation surfaces for the controller and CRDs.
- AppRole authentication using externally managed role ID and Secret ID Secrets.

## Non-goals for the first slice

- Managing or configuring OpenBao auth-method mounts, TokenReview configuration, or secret engines; the Kubernetes Auth mount and its platform configuration must exist in advance even when role objects are managed by this operator.
- Synchronizing secrets or tokens into Kubernetes.
- Managing entity merges or OIDC configuration.
- Promising compatibility with Vault distributions or with every OpenBao plugin API.
- Treating the generated OpenAPI document as a stable code-generation contract.

The future identity capabilities are captured as stories and backlog outcomes so the current API can evolve without making the first implementation broad or speculative.
