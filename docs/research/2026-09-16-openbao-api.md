# OpenBao API research — 2026-09-16

## Scope and source quality

This note covers the stable OpenBao v2.6.2 release available on 2026-09-16 and the official OpenBao API documentation. The generated OpenAPI file is a runtime snapshot, so its enabled mounts and release version are part of its provenance.

Primary sources:

- [OpenBao v2.6.2 release](https://github.com/openbao/openbao/releases/tag/v2.6.2)
- [Official OpenAPI generation script](https://github.com/openbao/openbao/blob/main/scripts/gen_openapi.sh)
- [OpenBao HTTP API documentation](https://github.com/openbao/openbao/blob/main/website/content/api-docs/index.mdx)
- [OpenBao namespaces](https://openbao.org/docs/concepts/namespaces/)
- [OpenBao namespace API](https://openbao.org/docs/next/api/system/namespaces/)
- [OpenBao identity entity API](https://openbao.org/docs/2.4.x/api/secret/identity/entity/)
- [OpenBao identity concepts](https://openbao.org/docs/next/concepts/identity/)
- [OpenBao Kubernetes Auth](https://openbao.org/docs/next/auth/kubernetes/)
- [OpenBao Kubernetes Auth (current documentation)](https://openbao.org/docs/auth/kubernetes/)
- [OpenBao AppRole](https://openbao.org/docs/auth/approle/)
- [OpenBao token auth API](https://openbao.org/docs/2.4.x/api/auth/token/)
- [OpenBao ACL policy API](https://openbao.org/docs/2.4.x/api/system/policies/)

## Observations

- OpenBao's HTTP API is currently v1 and prefixes routes with `/v1/`.
- Authenticated requests accept `X-Vault-Token` or an `Authorization: Bearer` header. The OpenBao CLI and SDK also send `X-Vault-Request: true`; the client follows that convention.
- Identity entity operations include create/update, read by ID, read by name, and delete by ID. The entity request supports `name`, `metadata`, `policies`, and `disabled`.
- Identity entity-alias operations include create, list IDs, read by ID, update by ID, and delete by ID. Alias requests use `canonical_id`, `mount_accessor`, and `name`; OpenBao does not expose a direct alias lookup by name and mount accessor.
- Identity group operations include create, read by ID, read by name, update by ID, and delete by ID. Group requests use `name`, `type`, `metadata`, `policies`, `member_entity_ids`, and `member_group_ids`.
- OpenBao models group membership as fields on the group rather than as an independent membership endpoint. Internal groups support direct entity and subgroup membership; external groups use an external alias to map membership managed outside the identity store.
- The health endpoint can use non-2xx status codes for standby, sealed, or uninitialized states; the response body still carries health information.
- OpenBao exposes OpenAPI through `/v1/sys/internal/specs/openapi`. The official script starts OpenBao, enables selected built-in plugins, and queries that endpoint with `generic_mount_paths`.
- OpenBao namespaces provide isolated identity and policy domains, including entities and groups. Namespace-aware API requests can use the `X-Vault-Namespace` header with an absolute or relative hierarchical namespace path; the root namespace is the default when the header is omitted.
- Namespace paths cannot end with `/`, contain spaces, or use reserved path segments such as `root`, `sys`, `auth`, `cubbyhole`, or `identity`.
- In the verified OpenBao v2.6.2 Kind instance, `GET /v1/sys/health` with a namespace header returned HTTP 400 (`operation unavailable in namespaces`), while namespaced token self-lookup remained available. The connection controller therefore uses health checks for root connections and namespaced token self-lookup for namespace-scoped readiness.
- OpenBao's Kubernetes Auth login endpoint is `POST /v1/auth/<mount>/login` (the default mount is `kubernetes`). It accepts a role and Kubernetes ServiceAccount JWT and returns an `auth.client_token` with lease duration and renewability metadata. The mount must be enabled and configured by the OpenBao administrator before the operator can use it.
- OpenBao's AppRole login endpoint is `POST /v1/auth/<mount>/login` (the default mount is `approle`). It accepts `role_id` and `secret_id` and returns an `auth.client_token` with lease duration and renewability metadata. The mount, role, and Secret ID lifecycle must be managed outside the operator.
- The token API exposes `POST /v1/auth/token/renew-self` for renewing the current token. The client uses that endpoint before a renewable lease expires and falls back to a fresh Kubernetes Auth login when renewal fails or a request is rejected.
- OpenBao ACL policies are managed through `GET`, `POST`, and `DELETE /v1/sys/policies/acl/:name`; the list endpoint is `/v1/sys/policies/acl`. Reads return the document in `data.policy` and include the policy name and version. Writes send the document in a `policy` request field.
- OpenBao Kubernetes Auth workload roles are managed below `auth/<mount>/role/<name>`. The official role example binds ServiceAccount names and namespaces and sets token policy and lifetime fields; the mount and Kubernetes TokenReview configuration must exist before role operations are useful. The operator maps the supported Kubernetes fields to the token-prefixed role wire keys and keeps mount enablement outside its scope.
- OpenBao exposes durable control-plane configuration through `/v1/sys/auth/:path`, `/v1/sys/mounts/:path`, `/v1/sys/namespaces/:path`, `/v1/sys/audit/:path`, `/v1/sys/quotas/rate-limit/:name`, `/v1/sys/workflows/manage/:path`, and `/v1/sys/plugins/catalog/:type/:name`. These are suitable for typed reconciliation because they have named records with read/write/delete contracts; the server hosting them remains external.
- OpenBao's identity OIDC configuration is represented by `/v1/identity/oidc/config` and named provider, client, key, role, scope, and assignment endpoints. The operator manages configuration fields but deliberately does not expose generated client secrets or execute authorization/token operations.
- OpenBao password policies, token roles, AppRole roles, and group aliases are durable named configuration records. Their credential issuance and login flows remain separate from configuration reconciliation.
- OpenBao exposes additional durable identity configuration through personas and MFA login-enforcement and provider-method endpoints. Duo, Okta, and PingID provider credentials are write-only inputs from the operator's perspective; the operator reads them from Kubernetes Secrets, hashes them for drift detection, and never stores them in status.
- OpenBao exposes durable system configuration through `/v1/sys/config/cors`, `/v1/sys/config/auditing/request-headers/:header`, `/v1/sys/config/ui/headers/:header`, `/v1/sys/quotas/config`, `/v1/sys/loggers` and `/v1/sys/loggers/:name`, `/v1/sys/rotate/config`, and `/v1/sys/rotate/keyring/config`. These are configuration endpoints rather than immediate administrative actions, so they are suitable for typed reconciliation.

## Generated reference

The checked-in [`hack/openbao-openapi.json`](../../hack/openbao-openapi.json) was generated from the verified OpenBao v2.6.2 Linux amd64 release binary using the official runtime endpoint. The release archive checksum was verified as:

```text
8dc11cc5fca0b539a9e352727dacb4e2d304daffcf9a66e0718ac325a20d05aa
```

The resulting document is OpenAPI 3.0.2, contains 231 paths and includes the identity entity, alias, group, and ACL policy endpoints used by this project, and has SHA-256:

```text
82f689cc39f60992e25f786c5737bb784f9bfe3c8c50006aa6cee8102aa2ec3d
```

## Inferences used by this project

- The operator binds an entity to the ID returned by OpenBao and uses name lookup only for initial adoption or creation. This avoids using a mutable name as the long-term identity.
- The operator binds an alias to the ID returned by OpenBao and uses the alias ID list plus candidate reads only for initial adoption. This avoids treating the mutable alias name/mount pair as the long-term identity.
- The operator binds a group to the ID returned by OpenBao and treats each Kubernetes membership as an explicit owned edge. Because OpenBao updates membership at group scope, the controller tracks previously managed member IDs in group status and preserves unclaimed remote edges.
- The OpenAPI snapshot is a semantic reference rather than a generator input because OpenBao generates it at runtime and it can vary with version and enabled mounts.
- The checked-in runtime OpenAPI snapshot does not include the Kubernetes Auth login route because the snapshot was captured before that plugin mount was enabled. The route is therefore tracked against the official auth documentation and covered by focused HTTP contract tests rather than added as an unavailable snapshot path.
- The checked-in runtime OpenAPI snapshot also does not include the AppRole login route because auth plugin routes are mount-dependent. The route is tracked against the official AppRole documentation and covered by focused HTTP contract tests rather than added as an unavailable snapshot path.
- The checked-in runtime OpenAPI snapshot does not include the Kubernetes Auth role route because auth plugin routes are mount-dependent. The role route and field behavior are tracked against the official Kubernetes Auth documentation and the verified OpenBao v2.6.2 Kind instance, then covered by focused HTTP contract tests and the narrow live role scenario.
- Typed clients are preferred for each durable OpenBao surface. The project does not expose an arbitrary-path CRD: a route is included only when desired state is durable, read/write behavior is observable, and credential or one-shot-operation semantics are safe for reconciliation.
- The operator models automatic-rotation configuration but deliberately does not call immediate rotation endpoints. A periodic configuration resource must not unexpectedly rotate an OpenBao key as a side effect of a normal reconcile.
- Namespace targeting belongs on `OpenBaoConnection` because it changes the API and credential context for every resource using that connection. The operator therefore validates and applies one header in the shared client rather than duplicating namespace handling across controllers.
- The user-facing policy field is named `rules` while the typed client maps it to OpenBao's `policy` field. This keeps the Kubernetes resource clear without hiding the OpenBao wire contract.
- A policy resource does not need a generated external ID: the OpenBao policy name is the immutable Kubernetes resource name, while status version and hash provide useful observed identity and drift evidence.

## Unresolved questions

- The OpenBao documentation warns that v1 compatibility is not yet promised. Future releases need focused contract tests before updating the reference.
- The operator manages OpenBao's durable configuration records broadly, but deliberately excludes OpenBao server deployment, storage and HA lifecycle, initialization and unseal workflows, credential issuance, arbitrary secret data, diagnostics, and one-shot administrative actions such as rekey or lease tidy.
