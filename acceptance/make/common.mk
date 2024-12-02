KIND_CLUSTER_NAME ?= namespace-lister-acceptance-tests
IMG ?= namespace-lister:latest

ROOT_DIR ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
LOCALBIN ?= $(ROOT_DIR)/bin

OUTDIR ?= $(ROOT_DIR)/out
TMPDIR ?= $(ROOT_DIR)/tmp

GO ?= go

GOLANG_CI ?= $(GO) run -modfile $(shell dirname $(ROOT_DIR))/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

KUBECONFIG ?=
KUBECTL ?= kubectl
KUBECTL_X := KUBECONFIG=$(KUBECONFIG) $(KUBECTL)
KIND ?= kind
KIND_X := KUBECONFIG=$(KUBECONFIG) $(KIND)
KUBECONFIG_ATSA ?= /tmp/namespace-lister-acceptance-tests-user.kcfg

## Local Folders
$(LOCALBIN):
	mkdir $(LOCALBIN)
$(OUTDIR):
	@mkdir $(OUTDIR)
$(TMPDIR):
	@mkdir $(TMPDIR)

.PHONY: lint
lint: ## Run go linter.
	$(GOLANG_CI) run ./...

.PHONY: image-build
image-build:
	$(MAKE) -C $(ROOT_DIR)/.. image-build

.PHONY: kind-create
kind-create:
	$(KIND_X) create cluster --name "$(KIND_CLUSTER_NAME)" --config kind-config.yaml

.PHONY: kind-load-image
kind-load-image:
	$(KIND_X) load docker-image --name "$(KIND_CLUSTER_NAME)" "$(IMG)"

.PHONY: update-namespace-lister
update-namespace-lister: image-build load-image
	$(KUBECTL_X) rollout restart deployment namespace-lister -n namespace-lister
	$(KUBECTL_X) rollout status deployment -n namespace-lister namespace-lister

.PHONY: deploy-test-infra
deploy-test-infra:
	$(KUBECTL_X) apply -k $(ROOT_DIR)/dependencies/cert-manager/
	sleep 5
	$(KUBECTL_X) wait --for=condition=Ready --timeout=300s -l 'app.kubernetes.io/instance=cert-manager' -n cert-manager pod
	$(KUBECTL_X) apply -k $(ROOT_DIR)/dependencies/cluster-issuer/

.PHONY: create-test-identity
create-test-identity:
	$(KUBECTL_X) apply -k $(ROOT_DIR)/config/acceptance-tests/

.PHONY: export-test-identity-kubeconfig
export-test-identity-kubeconfig:
	$(KIND_X) get kubeconfig --name $(KIND_CLUSTER_NAME) | \
		yq '.users[0].user={"token": "'$$($(KUBECTL_X) get secret acceptance-tests-user -n acceptance-tests -o jsonpath='{.data.token}' | base64 -d )'"}' >| \
		$(KUBECONFIG_ATSA)

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	$(KUBECTL_X) delete namespace -l 'namespace-lister/scope=acceptance-tests'

