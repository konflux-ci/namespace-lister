KIND_CLUSTER_NAME =? namespace-lister-acceptance-tests
IMG ?= namespace-lister:latest

ROOT_DIR ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
LOCALBIN ?= $(ROOT_DIR)/bin

OUTDIR ?= $(ROOT_DIR)/out
TMPDIR ?= $(ROOT_DIR)/tmp

GO ?= go

GOLANG_CI ?= $(GO) run -modfile $(shell dirname $(ROOT_DIR))/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

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
	kind create cluster --name "$(KIND_CLUSTER_NAME)" --config kind-config.yaml

.PHONY: load-image
load-image:
	kind load docker-image --name "$(KIND_CLUSTER_NAME)" "$(IMG)"

.PHONY: update-namespace-lister
update-namespace-lister: image-build load-image
	kubectl rollout restart deployment namespace-lister -n namespace-lister
	kubectl rollout status deployment -n namespace-lister namespace-lister

.PHONY: deploy-test-infra
deploy-test-infra:
	kubectl apply -k $(ROOT_DIR)/dependencies/cert-manager/
	sleep 5
	kubectl wait --for=condition=Ready --timeout=300s -l 'app.kubernetes.io/instance=cert-manager' -n cert-manager pod
	kubectl apply -k $(ROOT_DIR)/dependencies/cluster-issuer/

.PHONY: create-test-identity
create-test-identity:
	kubectl apply -k $(ROOT_DIR)/config/acceptance-tests/

.PHONY: export-test-identity-kubeconfig
export-test-identity-kubeconfig:
	kind get kubeconfig --name $(KIND_CLUSTER_NAME) | \
		yq '.users[0].user={"token": "'$$(kubectl get secret acceptance-tests-user -n acceptance-tests -o jsonpath='{.data.token}' | base64 -d )'"}' >| \
		/tmp/namespace-lister-acceptance-tests-user.kcfg

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	kubectl delete namespace -l 'namespace-lister/scope=acceptance-tests'

