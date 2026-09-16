# Product scope

## Promise

OpenBao Entity Operator lets a platform team declare OpenBao identity entities and groups as Kubernetes resources. It manages them through an authenticated OpenBao connection and exposes readiness, external identity, membership, and reconciliation failures through Kubernetes status.

## Current scope

- Same-namespace `OpenBaoConnection` resources with an address, optional OpenBao namespace, token Secret, optional CA bundle Secret, and request timeout.
- Same-namespace `OpenBaoEntity` resources whose Kubernetes name is the OpenBao entity name.
- Same-namespace `OpenBaoEntityAlias` resources that bind an auth-method alias name and mount accessor to an `OpenBaoEntity`.
- Same-namespace `OpenBaoGroup` resources for internal or external OpenBao identity groups.
- Same-namespace `OpenBaoGroupMembership` resources that claim one entity or subgroup relationship for an internal group.
- Create, adopt, update, observe, drift-correct, orphan, and opt-in delete behavior.
- Desired metadata, ACL policy names, and disabled state.
- Status conditions that distinguish dependencies, configuration, authentication, and external API failures.
- Helm and generated Kustomize installation surfaces for the controller and CRDs.

## Non-goals for the first slice

- Managing OpenBao auth-method mounts or secret engines.
- Synchronizing secrets or tokens into Kubernetes.
- Managing entity merges or OIDC configuration.
- Promising compatibility with Vault distributions or with every OpenBao plugin API.
- Treating the generated OpenAPI document as a stable code-generation contract.

The future identity capabilities are captured as stories and backlog outcomes so the current API can evolve without making the first implementation broad or speculative.
