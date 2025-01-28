ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
LOCALBIN := $(ROOT_DIR)/bin

OUT_DIR := $(ROOT_DIR)/out

GINKGO ?= ginkgo
GO ?= go

PERF_OUT_DIR := $(OUT_DIR)/perf
PERF_CLUSTER_PROVIDER ?= kwokctl
PERF_CLUSTER_KUBECONFIG ?= $(PERF_OUT_DIR)/namespace-lister-perf-test
PERF_CLUSTER_PROVIDER_FLAGS ?= --disable-qps-limits

GOLANG_CI ?= $(GO) run -modfile $(ROOT_DIR)/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

IMG ?= namespace-lister:latest
IMAGE_BUILDER ?= docker

## Local Folders
$(LOCALBIN):
	mkdir $(LOCALBIN)
$(OUT_DIR):
	@mkdir $(OUT_DIR)

.PHONY: clean
clean: ## Delete local folders.
	@-rm -r $(LOCALBIN)
	@-rm -r $(OUT_DIR)

.PHONY: lint
lint: lint-go lint-yaml ## Run all linters.

.PHONY: lint-go
lint-go: ## Run golangci-lint to lint go code.
	@$(GOLANG_CI) run ./...

.PHONY: lint-yaml
lint-yaml: ## Lint yaml manifests.
	@yamllint .

.PHONY: vet
vet: ## Run go vet against code.
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Run go tidy against code.
	$(GO) mod tidy

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GO) fmt ./...

.PHONY: test
test: ## Run go test against code.
	$(GINKGO) --label-filter='!perf'

.PHONY: test-perf
test-perf: ## Run performance tests
	-$(PERF_CLUSTER_PROVIDER) delete cluster --name namespace-lister-perf-test
	KUBECONFIG=$(PERF_CLUSTER_KUBECONFIG) $(PERF_CLUSTER_PROVIDER) create cluster \
		--name namespace-lister-perf-test $(PERF_CLUSTER_PROVIDER_FLAGS)
	KUBECONFIG=$(PERF_CLUSTER_KUBECONFIG) $(GINKGO) --label-filter='perf' \
		--keep-going --procs=1 --flake-attempts 2 --output-dir=$(PERF_OUT_DIR)

.PHONY: image-build
image-build:
	$(IMAGE_BUILDER) build -t "$(IMG)" .

.PHONY: generate-code
generate-code: mockgen  ## Run go generate on the project.
	@echo $(GO) generate ./...
	@PATH=$(LOCALBIN):${PATH} $(GO) generate ./...

.PHONY: mockgen
mockgen: $(LOCALBIN) ## Install mockgen locally.
	$(GO) build -modfile $(ROOT_DIR)/hack/tools/mockgen/go.mod -o $(LOCALBIN)/mockgen go.uber.org/mock/mockgen
