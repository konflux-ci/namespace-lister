KIND_CLUSTER_NAME ?= namespace-lister-acceptance-tests
IMG ?= namespace-lister:latest
IMAGE_BUILDER ?= docker

ROOT_DIR ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
LOCALBIN ?= $(ROOT_DIR)/bin

OUT_DIR ?= $(ROOT_DIR)/out

GO ?= go

GOLANG_CI ?= $(GO) run -modfile $(shell dirname $(ROOT_DIR))/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

KUBECTL ?= kubectl
KIND ?= kind
KUBECONFIG_ATSA ?= /tmp/namespace-lister-acceptance-tests-user.kcfg

## Local Folders
$(LOCALBIN):
	mkdir $(LOCALBIN)
$(OUT_DIR):
	@mkdir $(OUT_DIR)

.PHONY: lint
lint: ## Run go linter.
	$(GOLANG_CI) run ./...

.PHONY: image-build
image-build:
	$(MAKE) -C $(ROOT_DIR)/.. image-build

.PHONY: kind-create
kind-create:
	$(KIND) create cluster --name "$(KIND_CLUSTER_NAME)" --config kind-config.yaml

.PHONY: kind-load-image
kind-load-image:
	$(IMAGE_BUILDER) save $(IMG) | \
		$(KIND) load image-archive --name $(KIND_CLUSTER_NAME) /dev/stdin

.PHONY: update-namespace-lister
update-namespace-lister: image-build load-image
	$(KUBECTL) rollout restart deployment namespace-lister -n namespace-lister
	$(KUBECTL) rollout status deployment -n namespace-lister namespace-lister

.PHONY: deploy-test-infra
deploy-test-infra:
	$(KUBECTL) apply -k $(ROOT_DIR)/dependencies/cert-manager/
	$(KUBECTL) rollout status \
		--timeout=300s \
		-l 'app.kubernetes.io/instance=cert-manager' \
		-n cert-manager deployment
	@echo "Waiting for cert-manager webhook to accept connections..."
	@webhook_ready=false; \
	for i in $$(seq 1 30); do \
		if $(KUBECTL) get validatingwebhookconfigurations cert-manager-webhook >/dev/null 2>&1 && \
		   echo '{"apiVersion":"cert-manager.io/v1","kind":"Issuer","metadata":{"name":"probe","namespace":"default"},"spec":{"selfSigned":{}}}' | \
		   $(KUBECTL) apply --dry-run=server -f - >/dev/null 2>&1; then \
			webhook_ready=true; \
			break; \
		fi; \
		echo "  webhook not ready yet, retrying ($$i/30)..."; \
		sleep 2; \
	done; \
	if [ "$$webhook_ready" != "true" ]; then \
		echo "ERROR: cert-manager webhook did not become ready after 30 attempts"; \
		exit 1; \
	fi
	$(KUBECTL) apply -k $(ROOT_DIR)/dependencies/cluster-issuer/

.PHONY: create-test-identity
create-test-identity:
	$(KUBECTL) apply -k $(ROOT_DIR)/config/acceptance-tests/

.PHONY: export-test-identity-kubeconfig
export-test-identity-kubeconfig:
	$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) | \
		yq '.users[0].user={"token": "'$$($(KUBECTL) get secret acceptance-tests-user -n acceptance-tests -o jsonpath='{.data.token}' | base64 -d )'"}' >| \
		$(KUBECONFIG_ATSA)

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	$(KUBECTL) delete namespace -l 'namespace-lister/scope=acceptance-tests'

