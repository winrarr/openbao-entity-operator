# Compatibility policy

OpenBao Entity Operator follows a latest-stable policy. The project supports
the newest stable GA OpenBao release selected for the current project line; it
does not promise compatibility with every older release or with pre-release
builds.

As of 2026-09-16, the selected release is [OpenBao
v2.6.2](https://github.com/openbao/openbao/releases/tag/v2.6.2). The
`Makefile` pins the Kind image to `openbao/openbao:2.6.2`, and
`hack/openbao-openapi.json` is a runtime snapshot captured from that release.
The `v2.7.0-beta20260909` release is a pre-release and is not part of the
support promise.

## What compatibility means

- The selected stable release is the version exercised by `make kind-e2e` and
  used to produce the checked-in OpenAPI reference.
- Kubernetes API behavior, controller logic, and typed HTTP contracts are
  tested independently of the live server through unit and `httptest` tests.
- Older OpenBao releases may work when their endpoint contracts are unchanged,
  but that is incidental and not a supported compatibility guarantee.
- A new stable OpenBao release is adopted deliberately; the project does not
  silently follow a mutable `latest` container tag.

## Updating the selected release

When a newer stable release becomes the project target:

1. Verify the release and its security notes from the official OpenBao release
   sources.
2. Update `OPENBAO_IMAGE` in the root `Makefile`.
3. Recreate `hack/openbao-openapi.json` from the selected release with
   `make update-openbao-openapi OPENBAO_TOKEN=...` against a local instance.
4. Update the provenance and checksum in
   `docs/research/2026-09-16-openbao-api.md`.
5. Run `make check`, `make kind-e2e`, and `make verify-generated`.
6. Review endpoint changes and add focused client contract tests before
   changing controller behavior.

The compatibility decision should be recorded in the same change as the
version update. Do not add compatibility shims for older releases unless a
concrete supported deployment requires them and the behavior can be tested
without broadening the operator into a version-branching API client.
