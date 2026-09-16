# Local development

Use the repository Makefile as the canonical interface:

```sh
make check
make build
make helm-lint helm-template
make helm-package
make kind-e2e
make kind-down
```

The live target builds and loads the local operator image, installs the operator from the Helm chart, and runs the OpenBao workflow, so Docker, Helm, and Kind are required. It uses the default Kind CNI for the normal workflow. Use `KIND_CNI=cilium` when the scenario specifically needs Cilium behavior.

Do not commit `bin/`, `dist/`, coverage output, local kubeconfigs, Kind state, generated OpenBao tokens, or test credentials.
