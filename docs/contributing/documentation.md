# Documentation

Markdown under `docs/` is the source of truth for the documentation site. `zensical.toml` defines navigation, repository links, and strict validation. The generated CRD reference at `docs/reference/api.md` comes from the Go API definitions and `hack/crd-ref-docs.yaml`; edit API comments and regenerate it instead of editing that file directly.

Build the site locally with the pinned container image:

```sh
make docs-build
```

Serve it while writing:

```sh
make docs-serve
```

The docs workflow builds the site on pull requests and publishes it to GitHub Pages from `main`. Keep credentials, kubeconfigs, generated tokens, private endpoints, and local-cluster state out of documentation.
