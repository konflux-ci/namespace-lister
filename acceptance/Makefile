.PHONY: prepare
prepare: image-build kind-create load-image deploy-test-infra deploy-namespace-lister deploy-test-proxy
	@:

.PHONY: update-namespace-lister
update-namespace-lister: image-build load-image
	kubectl rollout restart deployment namespace-lister -n namespace-lister
	kubectl rollout status deployment -n namespace-lister namespace-lister

.PHONY: image-build
image-build:
	$(MAKE) -C .. image-build

.PHONY: kind-create
kind-create:
	kind create cluster --name namespace-lister --config kind-config.yaml

.PHONY: load-image
load-image:
	 kind load docker-image --name namespace-lister namespace-lister:latest 

.PHONY: deploy-test-infra
deploy-test-infra:
	kubectl apply -k ./dependencies/cert-manager/
	sleep 5
	kubectl wait --for=condition=Ready --timeout=300s -l 'app.kubernetes.io/instance=cert-manager' -n cert-manager pod
	kubectl apply -k ./dependencies/cluster-issuer/

.PHONY: deploy-test-proxy
deploy-test-proxy:
	kubectl apply -k ./config/proxy/

.PHONY: deploy-namespace-lister
deploy-namespace-lister:
	cd ../config/ && \
		kustomize edit set namespace namespace-lister
	kubectl apply -k ../config/

.PHONY: create-test-identity
create-test-identity:
	kubectl apply -k ./config/acceptance-tests/

.PHONY: export-test-identity-kubeconfig
export-test-identity-kubeconfig:
	kind get kubeconfig --name namespace-lister > /tmp/namespace-lister-acceptance-tests-user.kcfg
	yq -i '.users[0].user={"token": "'$$(kubectl get secret acceptance-tests-user -n acceptance-tests -o jsonpath='{.data.token}' | base64 -d )'"}' /tmp/namespace-lister-acceptance-tests-user.kcfg

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	kubectl delete namespace -l namespace-lister/scope=acceptance-tests

.PHONY: wip
wip: vet clean create-test-identity export-test-identity-kubeconfig
	kubectl rollout status deployment -n namespace-lister namespace-lister
	kubectl rollout status deployment -n namespace-lister namespace-lister-proxy
	KUBECONFIG=/tmp/namespace-lister-acceptance-tests-user.kcfg \
	KONFLUX_ADDRESS=https://localhost:10443 \
	E2E_USE_INSECURE_TLS=true \
		go test ./... -v --godog.tags=wip --godog.concurrency=1

.PHONY: test
test: vet clean create-test-identity export-test-identity-kubeconfig
	kubectl rollout status deployment -n namespace-lister namespace-lister
	kubectl rollout status deployment -n namespace-lister namespace-lister-proxy
	KUBECONFIG=/tmp/namespace-lister-acceptance-tests-user.kcfg \
	KONFLUX_ADDRESS=https://localhost:10443 \
	E2E_USE_INSECURE_TLS=true \
		go test ./... -v --godog.tags=~skip --godog.concurrency=1


ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
LOCALBIN := $(ROOT_DIR)/bin

OUTDIR := $(ROOT_DIR)/out

GO ?= go

GOLANG_CI ?= $(GO) run -modfile $(shell dirname $(ROOT_DIR))/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

## Local Folders
$(LOCALBIN):
	mkdir $(LOCALBIN)
$(OUTDIR):
	@mkdir $(OUTDIR)

.PHONY: lint
lint: ## Run go linter.
	$(GOLANG_CI) run ./...
