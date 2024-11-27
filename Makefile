ROOT_DIR := $(realpath $(firstword $(MAKEFILE_LIST)))
LOCALBIN := $(ROOT_DIR)/bin

OUTDIR := $(ROOT_DIR)/out

GO ?= go

GOLANG_CI ?= $(GO) run -modfile $(shell dirname $(ROOT_DIR))/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

IMG ?= namespace-lister:latest
IMAGE_BUILDER ?= docker

## Local Folders
$(LOCALBIN):
	mkdir $(LOCALBIN)
$(OUTDIR):
	@mkdir $(OUTDIR)

.PHONY: clean
clean: ## Delete local folders.
	@-rm -r $(LOCALBIN)
	@-rm -r $(OUTDIR)

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
	$(GO) test ./...

.PHONY: image-build
image-build:
	$(IMAGE_BUILDER) build -t "$(IMG)" .
