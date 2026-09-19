# Local workflow configuration for the root Makefile.
#
# Keep supported command overrides and pinned fixture/tool defaults here so the
# Makefile remains focused on workflow orchestration. Every `?=` value can be
# overridden on the command line when a local environment needs a different
# tool, image, cluster, namespace, or documentation setup.

LOCALBIN ?= $(CURDIR)/bin

# Supported command and deployment overrides.
IMG ?= ghcr.io/winrarr/openbao-entity-operator:dev
CONTAINER_TOOL ?= docker
KUBECTL ?= kubectl
HELM ?= helm
HELM_ARGS ?=
HELM_INSTALL_ARGS ?=
KIND ?= $(shell command -v kind 2>/dev/null || echo $(LOCALBIN)/kind)
PROJECT_NAME ?= openbao-entity-operator
KIND_CLUSTER ?= openbao-entity-operator
KIND_CNI ?= default
OPERATOR_NAMESPACE ?= openbao-entity-operator-system
OPENBAO_IMAGE ?= openbao/openbao:2.6.2
OPENBAO_NAMESPACE ?= openbao
OPENBAO_TOKEN_SECRET ?= openbao-dev-token
OPENBAO_ADDR ?= http://127.0.0.1:8200
OPENBAO_TOKEN ?=

# Documentation settings.
DOCS_CONTAINER_IMAGE ?= zensical/zensical:0.0.59
DOCS_CONFIG ?= zensical.toml

# Pinned local fixture values. Override these only when the local environment
# needs a different compatibility or test-fixture version.
KIND_NODE_IMAGE ?= kindest/node:v1.34.2
KIND_VERSION ?= v0.30.0
OPENBAO_OPENAPI_SPEC ?= hack/openbao-openapi.json
CILIUM_VERSION ?= v1.20.1
E2E_TEST_NAMESPACE ?= openbao-entity-operator-e2e
E2E_TENANT_B_NAMESPACE ?= openbao-entity-operator-e2e-b
E2E_OUTSIDE_NAMESPACE ?= openbao-entity-operator-outside
PERSISTENT_OPENBAO_NAMESPACE ?= openbao-entity-operator-persistent
PERSISTENT_OPENBAO_DEPLOYMENT ?= openbao-persistent
PERSISTENT_OPENBAO_SERVICE ?= openbao-persistent
PERSISTENT_OPENBAO_TLS_SECRET ?= openbao-persistent-tls
PERSISTENT_OPENBAO_BOOTSTRAP_SECRET ?= openbao-persistent-bootstrap
PERSISTENT_E2E_NAMESPACE ?= openbao-entity-operator-resilience

# Tool bootstrap settings.
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GO_TOOLCHAIN ?= go1.27.1
KUSTOMIZE_VERSION ?= v5.8.0
CRD_REF_DOCS_VERSION ?= v0.3.0
CONTROLLER_TOOLS_VERSION ?= v0.22.0
GOLANGCI_LINT_VERSION ?= v2.13.1

GO := GOTOOLCHAIN=$(GO_TOOLCHAIN) go
GOFMT := $(shell GOTOOLCHAIN=$(GO_TOOLCHAIN) go env GOROOT)/bin/gofmt
KIND_OS ?= linux
KIND_ARCH ?= $(shell $(GO) env GOARCH)
KIND_PLATFORM ?= $(KIND_OS)/$(KIND_ARCH)
