# Product scope

## Promise

OpenBao Entity Operator lets a platform team declare OpenBao identity entities as Kubernetes resources. It manages the lifecycle of an entity through an authenticated OpenBao connection and exposes readiness, external identity, and reconciliation failures through Kubernetes status.

## Current scope

- Same-namespace `OpenBaoConnection` resources with an address, token Secret, optional CA bundle Secret, and request timeout.
- Same-namespace `OpenBaoEntity` resources whose Kubernetes name is the OpenBao entity name.
- Same-namespace `OpenBaoEntityAlias` resources that bind an auth-method alias name and mount accessor to an `OpenBaoEntity`.
- Create, adopt, update, observe, drift-correct, orphan, and opt-in delete behavior.
- Desired metadata, ACL policy names, and disabled state.
- Status conditions that distinguish dependencies, configuration, authentication, and external API failures.

## Non-goals for the first slice

- Managing OpenBao auth-method mounts or secret engines.
- Synchronizing secrets or tokens into Kubernetes.
- Managing groups, group membership, entity merges, or OIDC configuration.
- Promising compatibility with Vault distributions or with every OpenBao plugin API.
- Treating the generated OpenAPI document as a stable code-generation contract.

The future identity capabilities are captured as stories and backlog outcomes so the current API can evolve without making the first implementation broad or speculative.
