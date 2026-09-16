# OpenBao API research — 2026-09-16

## Scope and source quality

This note covers the stable OpenBao v2.6.2 release available on 2026-09-16 and the official OpenBao API documentation. The generated OpenAPI file is a runtime snapshot, so its enabled mounts and release version are part of its provenance.

Primary sources:

- [OpenBao v2.6.2 release](https://github.com/openbao/openbao/releases/tag/v2.6.2)
- [Official OpenAPI generation script](https://github.com/openbao/openbao/blob/main/scripts/gen_openapi.sh)
- [OpenBao HTTP API documentation](https://github.com/openbao/openbao/blob/main/website/content/api-docs/index.mdx)
- [OpenBao identity entity API](https://openbao.org/docs/2.4.x/api/secret/identity/entity/)

## Observations

- OpenBao's HTTP API is currently v1 and prefixes routes with `/v1/`.
- Authenticated requests accept `X-Vault-Token` or an `Authorization: Bearer` header. The OpenBao CLI and SDK also send `X-Vault-Request: true`; the client follows that convention.
- Identity entity operations include create/update, read by ID, read by name, and delete by ID. The entity request supports `name`, `metadata`, `policies`, and `disabled`.
- Identity entity-alias operations include create, list IDs, read by ID, update by ID, and delete by ID. Alias requests use `canonical_id`, `mount_accessor`, and `name`; OpenBao does not expose a direct alias lookup by name and mount accessor.
- The health endpoint can use non-2xx status codes for standby, sealed, or uninitialized states; the response body still carries health information.
- OpenBao exposes OpenAPI through `/v1/sys/internal/specs/openapi`. The official script starts OpenBao, enables selected built-in plugins, and queries that endpoint with `generic_mount_paths`.

## Generated reference

The checked-in [`hack/openbao-openapi.json`](../../hack/openbao-openapi.json) was generated from the verified OpenBao v2.6.2 Linux amd64 release binary using the official runtime endpoint. The release archive checksum was verified as:

```text
8dc11cc5fca0b539a9e352727dacb4e2d304daffcf9a66e0718ac325a20d05aa
```

The resulting document is OpenAPI 3.0.2, contains 231 paths and 10 identity-entity paths, and has SHA-256:

```text
82f689cc39f60992e25f786c5737bb784f9bfe3c8c50006aa6cee8102aa2ec3d
```

## Inferences used by this project

- The operator binds an entity to the ID returned by OpenBao and uses name lookup only for initial adoption or creation. This avoids using a mutable name as the long-term identity.
- The operator binds an alias to the ID returned by OpenBao and uses the alias ID list plus candidate reads only for initial adoption. This avoids treating the mutable alias name/mount pair as the long-term identity.
- The OpenAPI snapshot is a semantic reference rather than a generator input because OpenBao generates it at runtime and it can vary with version and enabled mounts.
- A small typed client is sufficient for the current entity and connection stories and keeps unrelated secret-engine APIs outside the initial dependency surface.

## Unresolved questions

- The OpenBao documentation warns that v1 compatibility is not yet promised. Future releases need focused contract tests before updating the reference.
- The current alias resource models one auth mount accessor per alias. Group membership and more complex multi-mount workflows remain future design stories, not hidden current requirements.
