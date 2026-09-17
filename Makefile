SHELL := /usr/bin/env bash
.SHELLFLAGS := -o pipefail -ec
.DEFAULT_GOAL := help

PROJECT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCALBIN ?= $(PROJECT_DIR)/bin

IMG ?= ghcr.io/winrarr/openbao-entity-operator:dev
CONTAINER_TOOL ?= docker
DOCS_CONTAINER_IMAGE ?= zensical/zensical:0.0.59
DOCS_CONTAINER_MOUNTS = -v "$(PROJECT_DIR)":/docs
DOCS_CONFIG ?= zensical.toml
KUBECTL ?= kubectl
KUBECTL_ARGS ?=
KUBECTL_CMD = $(KUBECTL) $(KUBECTL_ARGS)
KIND ?= $(shell command -v kind 2>/dev/null || echo $(LOCALBIN)/kind)
HELM ?= helm
HELM_ARGS ?=
HELM_CMD = $(HELM) $(HELM_ARGS)
HELM_INSTALL_ARGS ?=
PROJECT_NAME ?= openbao-entity-operator
CHART_DIR ?= charts/openbao-entity-operator
CHART_PACKAGE_DIR ?= dist
KIND_CLUSTER ?= openbao-entity-operator
KIND_CNI ?= default
KIND_NODE_IMAGE ?= kindest/node:v1.34.2
OPENBAO_IMAGE ?= openbao/openbao:2.6.2
OPENBAO_NAMESPACE ?= openbao
OPENBAO_TOKEN_SECRET ?= openbao-dev-token
OPENBAO_TOKEN_KEY ?= token
OPERATOR_NAMESPACE ?= openbao-entity-operator-system
E2E_TEST_NAMESPACE ?= openbao-entity-operator-e2e
E2E_TENANT_B_NAMESPACE ?= openbao-entity-operator-e2e-b
E2E_OUTSIDE_NAMESPACE ?= openbao-entity-operator-outside
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
OPENBAO_OPENAPI_SPEC ?= hack/openbao-openapi.json
OPENBAO_ADDR ?= http://127.0.0.1:8200
OPENBAO_TOKEN ?=

GO_TOOLCHAIN ?= go1.27.1
GO := GOTOOLCHAIN=$(GO_TOOLCHAIN) go
GOFMT := $(shell GOTOOLCHAIN=$(GO_TOOLCHAIN) go env GOROOT)/bin/gofmt
KUSTOMIZE_VERSION ?= v5.8.0
CRD_REF_DOCS_VERSION ?= v0.3.0
CONTROLLER_TOOLS_VERSION ?= v0.22.0
GOLANGCI_LINT_VERSION ?= v2.13.1
KIND_VERSION ?= v0.30.0
CILIUM_VERSION ?= v1.20.1

.PHONY: all
all: check build ## Run verification and build the manager.

##@ Help

.PHONY: help
help: ## Display available targets.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: format
format: ## Format Go sources.
	$(GO) fmt ./...

.PHONY: format-check
format-check: ## Fail when Go sources are not gofmt-clean.
	@files="$$($(GOFMT) -l $$(find api cmd internal -name '*.go' -type f))"; \
	if [ -n "$$files" ]; then echo "Go sources need formatting:" >&2; echo "$$files" >&2; exit 1; fi

.PHONY: generate
generate: controller-gen crd-ref-docs ## Generate Go and documentation artifacts.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt",year=$(shell date +%Y) paths="./..."
	$(MAKE) generate-api-reference

.PHONY: manifests
manifests: controller-gen ## Generate CRDs and RBAC from API and controller markers.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases
	$(MAKE) sync-chart-generated

.PHONY: sync-chart-generated
sync-chart-generated: ## Sync generated CRDs and manager RBAC to the Helm chart.
	./hack/sync-chart-crds.sh
	./hack/sync-chart-rbac.sh

.PHONY: verify-generated
verify-generated: manifests generate ## Verify generated files are current in a tracked checkout.
	@if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
		git diff --exit-code -- api/openbao/v1alpha1/zz_generated.deepcopy.go config/crd/bases config/rbac/role.yaml charts/$(PROJECT_NAME)/crds charts/$(PROJECT_NAME)/templates/clusterrole.yaml charts/$(PROJECT_NAME)/templates/rbac-helpers.yaml docs/reference/api.md; \
	else \
		echo "No Git checkout detected; generated files were regenerated but cannot be compared"; \
	fi

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: test
test: manifests generate format-check vet ## Run unit tests.
	$(GO) test ./... -coverprofile=cover.out

.PHONY: lint-config
lint-config: golangci-lint ## Validate the linter configuration.
	"$(GOLANGCI_LINT)" config verify

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	"$(GOLANGCI_LINT)" run

.PHONY: helm-lint
helm-lint: ## Lint the operator Helm chart.
	$(HELM) lint $(CHART_DIR)

.PHONY: helm-template
helm-template: ## Render the operator Helm chart.
	$(HELM) template $(PROJECT_NAME) $(CHART_DIR) --namespace $(OPERATOR_NAMESPACE) --include-crds >/dev/null

.PHONY: openapi-check
openapi-check: ## Validate the checked-in OpenBao OpenAPI reference.
	OPENBAO_OPENAPI_SPEC="$(OPENBAO_OPENAPI_SPEC)" ./hack/check-openbao-openapi.sh

.PHONY: shell-check
shell-check: ## Validate repository shell scripts parse successfully.
	bash -n hack/*.sh

.PHONY: update-openbao-openapi
update-openbao-openapi: ## Refresh the OpenBao OpenAPI reference from a running instance.
	@test -n "$(OPENBAO_TOKEN)" || { echo "Set OPENBAO_TOKEN to refresh the OpenBao OpenAPI reference" >&2; exit 1; }
	OPENBAO_ADDR="$(OPENBAO_ADDR)" OPENBAO_TOKEN="$(OPENBAO_TOKEN)" OPENBAO_OPENAPI_SPEC="$(OPENBAO_OPENAPI_SPEC)" ./hack/update-openbao-openapi.sh

.PHONY: kustomize-build
kustomize-build: kustomize ## Render the default installation manifests.
	"$(KUSTOMIZE)" build config/default >/dev/null

.PHONY: generate-api-reference
generate-api-reference: crd-ref-docs ## Generate the CRD API reference.
	@mkdir -p docs/reference
	"$(CRD_REF_DOCS)" \
		--config hack/crd-ref-docs.yaml \
		--renderer markdown \
		--source-path ./api \
		--output-path docs/reference/api.md
	@awk '{ lines[NR] = $$0 } END { last = NR; while (last > 0 && lines[last] == "") last--; for (i = 1; i <= last; i++) print lines[i] }' docs/reference/api.md > docs/reference/api.md.tmp
	@mv docs/reference/api.md.tmp docs/reference/api.md

.PHONY: build-docs-site
build-docs-site: generate-api-reference ## Build the documentation site with strict link validation.
	$(CONTAINER_TOOL) run --rm --workdir /docs $(DOCS_CONTAINER_MOUNTS) $(DOCS_CONTAINER_IMAGE) build --strict --config-file $(DOCS_CONFIG)

.PHONY: docs-build
docs-build: build-docs-site ## Generate the API reference and build the documentation site.

.PHONY: docs-serve
docs-serve: generate-api-reference ## Generate and serve the documentation site locally.
	$(CONTAINER_TOOL) run --rm --workdir /docs -p 8000:8000 $(DOCS_CONTAINER_MOUNTS) $(DOCS_CONTAINER_IMAGE) serve --dev-addr 0.0.0.0:8000 --config-file $(DOCS_CONFIG)

.PHONY: check
check: manifests generate format-check shell-check vet test lint-config lint helm-lint helm-template openapi-check kustomize-build docs-build ## Run the complete local verification suite.

##@ Build

.PHONY: build
build: manifests generate format-check vet ## Build the controller binary.
	@mkdir -p bin
	$(GO) build -trimpath -ldflags="-s -w" -o bin/manager ./cmd

.PHONY: run
run: manifests generate ## Run the controller against the current kubeconfig context.
	$(GO) run ./cmd

.PHONY: docker-build
docker-build: ## Build the controller image.
	$(CONTAINER_TOOL) build --provenance=false --sbom=false --tag $(IMG) .

.PHONY: docker-push
docker-push: ## Push the controller image.
	$(CONTAINER_TOOL) push $(IMG)

.PHONY: build-installer
build-installer: manifests generate kustomize ## Build a standalone Kustomize installation bundle.
	@mkdir -p dist
	"$(KUSTOMIZE)" build config/default | sed 's#image: controller:latest#image: $(IMG)#' > dist/install.yaml

.PHONY: helm-package
helm-package: manifests helm-lint ## Package the operator Helm chart.
	@mkdir -p "$(CHART_PACKAGE_DIR)"
	"$(HELM)" package "$(CHART_DIR)" --destination "$(CHART_PACKAGE_DIR)"

##@ Kubernetes deployment

.PHONY: install
install: manifests kustomize ## Install CRDs in the current Kubernetes context.
	"$(KUSTOMIZE)" build config/crd | $(KUBECTL_CMD) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Remove CRDs from the current Kubernetes context.
	"$(KUSTOMIZE)" build config/crd | $(KUBECTL_CMD) delete --ignore-not-found=true -f -

.PHONY: deploy
deploy: manifests generate helm-lint ## Install or upgrade the operator Helm chart.
	$(MAKE) helm-install

.PHONY: helm-install
helm-install: helm-lint ## Install or upgrade the operator Helm chart.
	IMG_REF="$(IMG)"; \
	if [[ "$$IMG_REF" == *@* ]]; then \
		IMG_REPO="$${IMG_REF%@*}"; \
		IMG_DIGEST="$${IMG_REF#*@}"; \
		$(HELM_CMD) upgrade --install "$(PROJECT_NAME)" "$(CHART_DIR)" \
			--namespace "$(OPERATOR_NAMESPACE)" --create-namespace \
			--set-string "image.repository=$$IMG_REPO" --set-string "image.digest=$$IMG_DIGEST" --set-string image.tag="" \
			$(HELM_INSTALL_ARGS) \
			--wait --timeout 5m; \
	else \
		IMG_LAST="$${IMG_REF##*/}"; \
		if [[ "$$IMG_LAST" == *:* ]]; then \
			IMG_REPO="$${IMG_REF%:*}"; \
			IMG_TAG="$${IMG_REF##*:}"; \
		else \
			IMG_REPO="$$IMG_REF"; \
			IMG_TAG="latest"; \
		fi; \
		$(HELM_CMD) upgrade --install "$(PROJECT_NAME)" "$(CHART_DIR)" \
			--namespace "$(OPERATOR_NAMESPACE)" --create-namespace \
			--set-string "image.repository=$$IMG_REPO" --set-string "image.tag=$$IMG_TAG" \
			$(HELM_INSTALL_ARGS) \
			--wait --timeout 5m; \
	fi

.PHONY: undeploy
undeploy: ## Uninstall the operator Helm release.
	$(HELM_CMD) uninstall "$(PROJECT_NAME)" --namespace "$(OPERATOR_NAMESPACE)" --ignore-not-found

##@ Local Kind environment

.PHONY: kind-up
kind-up: kind-install-openbao ## Create the disposable Kind cluster and OpenBao instance.

.PHONY: kind-create
kind-create: kind ## Create the named Kind cluster if it does not exist.
	@case "$(KIND_CNI)" in \
		default) kind_config=hack/kind-configuration.yaml ;; \
		cilium) kind_config=hack/kind-configuration-cilium.yaml ;; \
		*) echo "KIND_CNI must be 'default' or 'cilium'" >&2; exit 1 ;; \
	esac; \
	if ! "$(KIND)" get clusters | grep -Fxq "$(KIND_CLUSTER)"; then \
		"$(KIND)" create cluster --name "$(KIND_CLUSTER)" --image "$(KIND_NODE_IMAGE)" --config "$$kind_config"; \
	else \
		echo "Kind cluster $(KIND_CLUSTER) already exists"; \
	fi

.PHONY: kind-ensure-cni
kind-ensure-cni: kind-create ## Verify the named Kind cluster uses the selected CNI.
	@case "$(KIND_CNI)" in \
		default) \
			if "$(KUBECTL)" --context="kind-$(KIND_CLUSTER)" -n kube-system get daemonset kindnet >/dev/null 2>&1; then \
				echo "Using Kind's default CNI"; \
			else \
				echo "Kind cluster $(KIND_CLUSTER) was not created with the default CNI; run make kind-down before switching modes" >&2; exit 1; \
			fi ;; \
		cilium) \
			if "$(KUBECTL)" --context="kind-$(KIND_CLUSTER)" -n kube-system get daemonset kindnet >/dev/null 2>&1; then \
				echo "Kind cluster $(KIND_CLUSTER) still has the default CNI; run make kind-down before switching to Cilium" >&2; exit 1; \
			else \
				echo "Kind cluster $(KIND_CLUSTER) is configured for Cilium"; \
			fi ;; \
		*) echo "KIND_CNI must be 'default' or 'cilium'" >&2; exit 1 ;; \
	esac

.PHONY: kind-install-cni
kind-install-cni: kind-ensure-cni ## Install the selected CNI when required.
	@case "$(KIND_CNI)" in \
		default) ;; \
		cilium) $(MAKE) kind-install-cilium ;; \
		*) echo "KIND_CNI must be 'default' or 'cilium'" >&2; exit 1 ;; \
	esac

.PHONY: kind-install-cilium
kind-install-cilium: ## Install the pinned Cilium release into Kind.
	@"$(HELM)" upgrade --install cilium oci://quay.io/cilium/charts/cilium \
		--version "$(CILIUM_VERSION)" --namespace kube-system \
		--kube-context "kind-$(KIND_CLUSTER)" \
		--set kubeProxyReplacement=true \
		--set k8sServiceHost="$(KIND_CLUSTER)-control-plane" \
		--set k8sServicePort=6443 \
		--set operator.replicas=1 \
		--set hubble.enabled=true \
		--set hubble.relay.enabled=true \
		--set hubble.ui.enabled=false \
		--wait --timeout=10m

.PHONY: kind-load-openbao-image
kind-load-openbao-image: kind-ensure-cni ## Load the OpenBao image into Kind.
	@if ! "$(CONTAINER_TOOL)" image inspect "$(OPENBAO_IMAGE)" >/dev/null 2>&1; then \
		"$(CONTAINER_TOOL)" pull "$(OPENBAO_IMAGE)"; \
	fi
	@node_image="$(OPENBAO_IMAGE)"; \
	case "$$node_image" in */*/*|*.*/*|*:*/*) ;; *) node_image="docker.io/$$node_image" ;; esac; \
	if "$(CONTAINER_TOOL)" exec "$(KIND_CLUSTER)-control-plane" ctr --namespace=k8s.io images inspect "$$node_image" >/dev/null 2>&1; then \
		echo "OpenBao image $$node_image is already present on the Kind node"; \
	elif ! "$(KIND)" load docker-image "$(OPENBAO_IMAGE)" --name "$(KIND_CLUSTER)"; then \
		node_image="$(OPENBAO_IMAGE)"; \
		case "$$node_image" in */*/*|*.*/*|*:*/*) ;; *) node_image="docker.io/$$node_image" ;; esac; \
		echo "Kind image archive import failed; pulling $(OPENBAO_IMAGE) directly in the node" >&2; \
		"$(CONTAINER_TOOL)" exec "$(KIND_CLUSTER)-control-plane" ctr --namespace=k8s.io images pull --platform "$(KIND_PLATFORM)" "$$node_image" >/dev/null; \
	fi

.PHONY: kind-install-openbao
kind-install-openbao: kind-install-cni kind-load-openbao-image ## Install the disposable OpenBao dev server.
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" create namespace "$(OPENBAO_NAMESPACE)" --dry-run=client -o yaml | $(KUBECTL) --context="kind-$(KIND_CLUSTER)" apply -f -
	@if ! $(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPENBAO_NAMESPACE)" get secret "$(OPENBAO_TOKEN_SECRET)" >/dev/null 2>&1; then \
		token="$$(openssl rand -hex 32)"; \
		$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPENBAO_NAMESPACE)" create secret generic "$(OPENBAO_TOKEN_SECRET)" \
			--from-literal="$(OPENBAO_TOKEN_KEY)=$$token" --dry-run=client -o yaml | \
			$(KUBECTL) --context="kind-$(KIND_CLUSTER)" apply -f -; \
	fi
	OPENBAO_IMAGE="$(OPENBAO_IMAGE)" OPENBAO_NAMESPACE="$(OPENBAO_NAMESPACE)" OPENBAO_TOKEN_SECRET="$(OPENBAO_TOKEN_SECRET)" OPENBAO_TOKEN_KEY="$(OPENBAO_TOKEN_KEY)" \
		envsubst < hack/kind-openbao.yaml | $(KUBECTL) --context="kind-$(KIND_CLUSTER)" apply -f -
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPENBAO_NAMESPACE)" rollout status deployment/openbao --timeout=5m

.PHONY: kind-load-image
kind-load-image: kind-create docker-build ## Load the operator image into Kind.
	"$(KIND)" load docker-image "$(IMG)" --name "$(KIND_CLUSTER)"

.PHONY: kind-deploy
kind-deploy: ## Build and deploy the operator into Kind.
	$(MAKE) kind-up
	$(MAKE) kind-load-image
	$(MAKE) KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(KIND_CLUSTER)" install deploy
	$(MAKE) kind-restart

.PHONY: kind-deploy-e2e
kind-deploy-e2e: kind-deploy ## Build and deploy the operator for live E2E tests.
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" create namespace "$(E2E_TEST_NAMESPACE)" --dry-run=client -o yaml | $(KUBECTL) --context="kind-$(KIND_CLUSTER)" apply -f -
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" create namespace "$(E2E_TENANT_B_NAMESPACE)" --dry-run=client -o yaml | $(KUBECTL) --context="kind-$(KIND_CLUSTER)" apply -f -
	$(MAKE) HELM_INSTALL_ARGS="--set-string watchNamespaces[0]=$(E2E_TEST_NAMESPACE) --set-string watchNamespaces[1]=$(E2E_TENANT_B_NAMESPACE)" KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(KIND_CLUSTER)" helm-install

.PHONY: kind-e2e
kind-e2e: ## Run the live OpenBao reconciliation workflow in Kind.
	$(MAKE) kind-deploy-e2e
	KEEP_TEST_RESOURCES=true KUBECTL="$(KUBECTL)" KUBE_CONTEXT="kind-$(KIND_CLUSTER)" TEST_NAMESPACE="$(E2E_TEST_NAMESPACE)" TENANT_B_NAMESPACE="$(E2E_TENANT_B_NAMESPACE)" OUTSIDE_NAMESPACE="$(E2E_OUTSIDE_NAMESPACE)" OPENBAO_NAMESPACE="$(OPENBAO_NAMESPACE)" \
		OPENBAO_TOKEN_SECRET="$(OPENBAO_TOKEN_SECRET)" OPENBAO_TOKEN_KEY="$(OPENBAO_TOKEN_KEY)" OPERATOR_NAMESPACE="$(OPERATOR_NAMESPACE)" \
		OPERATOR_DEPLOYMENT="$(PROJECT_NAME)" \
		./hack/e2e-kind.sh
	KUBECTL="$(KUBECTL)" KUBE_CONTEXT="kind-$(KIND_CLUSTER)" TEST_NAMESPACE="$(E2E_TEST_NAMESPACE)" TENANT_B_NAMESPACE="$(E2E_TENANT_B_NAMESPACE)" OPENBAO_NAMESPACE="$(OPENBAO_NAMESPACE)" \
		OPERATOR_NAMESPACE="$(OPERATOR_NAMESPACE)" OPERATOR_DEPLOYMENT="$(PROJECT_NAME)" \
		./hack/e2e-multi-tenancy.sh
	$(MAKE) KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" kind-e2e-clean
	$(MAKE) HELM_INSTALL_ARGS="" KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(KIND_CLUSTER)" helm-install

.PHONY: kind-e2e-clean
kind-e2e-clean: kind ## Remove only the failed live E2E test resources.
	KUBECTL="$(KUBECTL)" KUBE_CONTEXT="kind-$(KIND_CLUSTER)" TEST_NAMESPACE="$(E2E_TEST_NAMESPACE)" TENANT_B_NAMESPACE="$(E2E_TENANT_B_NAMESPACE)" OUTSIDE_NAMESPACE="$(E2E_OUTSIDE_NAMESPACE)" ./hack/cleanup-kind-e2e.sh

.PHONY: kind-refresh
kind-refresh: ## Rebuild and redeploy the operator in Kind.
	$(MAKE) docker-build
	$(MAKE) kind-load-image
	$(MAKE) KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(KIND_CLUSTER)" deploy
	$(MAKE) kind-restart

.PHONY: kind-restart
kind-restart: ## Restart the operator after loading a mutable local image tag.
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPERATOR_NAMESPACE)" rollout restart deployment/$(PROJECT_NAME)
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPERATOR_NAMESPACE)" rollout status deployment/$(PROJECT_NAME) --timeout=5m

.PHONY: kind-down
kind-down: kind ## Delete only the named disposable Kind cluster.
	"$(KIND)" delete cluster --name "$(KIND_CLUSTER)"

##@ Dependencies

$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download the pinned Kustomize binary when needed.

$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download the pinned controller-gen binary when needed.

$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download the pinned golangci-lint binary when needed.

$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

$(CRD_REF_DOCS): $(LOCALBIN)
	$(call go-install-tool,$(CRD_REF_DOCS),github.com/elastic/crd-ref-docs,$(CRD_REF_DOCS_VERSION))

.PHONY: crd-ref-docs
crd-ref-docs: $(CRD_REF_DOCS) ## Download the pinned CRD reference generator.

KIND_OS ?= linux
KIND_ARCH ?= $(shell $(GO) env GOARCH)
KIND_PLATFORM ?= $(KIND_OS)/$(KIND_ARCH)

.PHONY: kind
kind: ## Use an installed Kind binary or download the pinned fallback.
	@if [ -x "$(KIND)" ] || command -v "$(KIND)" >/dev/null 2>&1; then \
		echo "Using Kind $$("$(KIND)" version 2>/dev/null | head -1)"; \
	else \
		echo "Downloading kind $(KIND_VERSION)"; \
		mkdir -p "$$(dirname "$(KIND)")"; \
		curl --fail --location --silent --show-error \
			-o "$(KIND).tmp" "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(KIND_OS)-$(KIND_ARCH)"; \
		chmod +x "$(KIND).tmp"; \
		mv "$(KIND).tmp" "$(KIND)"; \
	fi

define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
	set -e; \
	package=$(2)@$(3); \
	echo "Downloading $${package}"; \
	GOBIN="$(LOCALBIN)" $(GO) install $${package}; \
	mv "$(LOCALBIN)/$$(basename "$(1)")" "$(1)-$(3)"; \
}; \
ln -sf "$$(realpath "$(1)-$(3)")" "$(1)"
endef
