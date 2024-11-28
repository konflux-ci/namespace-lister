include ../../make/*.mk

CWD ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR := $(CWD)/../..
TMPDIR := $(CWD)/tmp
KIND_CLUSTER_NAME := namespace-lister-no-proxy

.PHONY: prepare
prepare: image-build kind-create load-image deploy-test-infra deploy-namespace-lister deploy-test-proxy
	@:

.PHONY: deploy-test-proxy
deploy-test-proxy:
	kubectl apply -k $(CWD)/config/proxy/

.PHONY: deploy-namespace-lister
deploy-namespace-lister: $(TMPDIR)
	-rm -r $(TMPDIR)/config
	mkdir -p $(TMPDIR)/config
	( \
		cd $(TMPDIR)/config && \
			kustomize init && \
			kustomize edit add base "../../../../../config" && \
			kustomize edit set namespace namespace-lister && \
			kustomize edit set image "namespace-lister:latest=$(IMG)" && \
			kustomize build | kubectl apply -f - ; \
	)

.PHONY: wip
wip: vet clean create-test-identity export-test-identity-kubeconfig
	kubectl rollout status deployment -n namespace-lister namespace-lister
	kubectl rollout status deployment -n namespace-lister namespace-lister-proxy
	KUBECONFIG=/tmp/namespace-lister-acceptance-tests-user.kcfg \
	KONFLUX_ADDRESS=https://localhost:10443 \
	E2E_USE_INSECURE_TLS=true \
		go test $(CWD)/... -v --godog.tags=wip --godog.concurrency=1

.PHONY: test
test: vet clean create-test-identity export-test-identity-kubeconfig
	kubectl rollout status deployment -n namespace-lister namespace-lister
	kubectl rollout status deployment -n namespace-lister namespace-lister-proxy
	KUBECONFIG=/tmp/namespace-lister-acceptance-tests-user.kcfg \
	KONFLUX_ADDRESS=https://localhost:10443 \
	E2E_USE_INSECURE_TLS=true \
		go test $(CWD)/... -v --godog.tags=~skip --godog.concurrency=1
