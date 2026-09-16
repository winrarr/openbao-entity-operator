# Releases

Release artifacts are published to GitHub Container Registry. The Helm chart
and operator image use the same stable version for the initial release line:

- `charts/openbao-entity-operator/Chart.yaml:version` selects the chart package;
- `charts/openbao-entity-operator/Chart.yaml:appVersion` selects the operator image.

## Package locally

Validate and package the chart into `dist/` with:

```sh
make helm-package
```

The package includes the CRDs under `charts/openbao-entity-operator/crds/`.
Run `make verify-generated` before packaging when changing API types or RBAC
markers.

## Release tags

Release tags are the publication interface:

- `vX.Y.Z` or `vX.Y.Z-rc.N` publishes the operator image and a matching chart;
- `chart-vX.Y.Z` or `chart-vX.Y.Z-rc.N` publishes a chart-only release.

For a paired release, both `version` and `appVersion` in `Chart.yaml` must
match the version in the `v*` tag. For a chart-only release, only `version`
must match the `chart-v*` tag; `appVersion` remains the already-published
operator image version.

The `Publish Release` workflow builds and publishes
`ghcr.io/winrarr/openbao-entity-operator:<version>`, pushes the chart to
`oci://ghcr.io/winrarr/charts/openbao-entity-operator`, and attaches the chart
package to the GitHub Release. A chart-only release verifies that its
`appVersion` image already exists before publishing.

To validate an existing tag without publishing anything:

```sh
gh workflow run publish-release.yaml -f tag_name=v0.1.0 -f dry_run=true
```

If publication fails after the tag is created, rerun the workflow for that tag
or dispatch it manually with the existing tag. Do not move or recreate a
release tag.
