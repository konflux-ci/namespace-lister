CWD ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR := $(CWD)/../..
TMPDIR := $(CWD)/tmp
KIND_CLUSTER_NAME := namespace-lister-smart-proxy
KUBECONFIG_ATSA := $(TMPDIR)/namespace-lister-acceptance-tests-user.kcfg
KUBECONFIG_ADMIN := $(TMPDIR)/namespace-lister-admin.kcfg
KONFLUX_ADDRESS := https://localhost:11443

KUBECONFIG := $(KUBECONFIG_ADMIN)
include ../../make/*.mk

.PHONY: prepare
prepare: image-build kind-create kind-load-image deploy-test-infra deploy-namespace-lister deploy-test-proxy
	@:

.PHONY: deploy-test-proxy
deploy-test-proxy:
	$(KUBECTL_X) apply -k ./config/proxy-auth/

.PHONY: deploy-namespace-lister
deploy-namespace-lister: $(TMPDIR)
	-rm -r $(TMPDIR)/config
	mkdir -p $(TMPDIR)/config
	( \
		cd $(TMPDIR)/config && \
			kustomize init && \
			kustomize edit add base "../../../../../config" && \
			cp "$(ROOT_DIR)/../config/patches/with-header-auth.yaml" . && \
			kustomize edit add patch --path "with-header-auth.yaml" --group "apps" --kind "Deployment" --name "namespace-lister" --version "v1"\
			kustomize edit set namespace namespace-lister && \
			kustomize edit set image "namespace-lister:latest=$(IMG)" && \
			kustomize build | $(KUBECTL_X) apply -f - ; \
	)

.PHONY: wip
wip: vet clean create-test-identity export-test-identity-kubeconfig
	$(KUBECTL_X) rollout status deployment -n namespace-lister namespace-lister
	$(KUBECTL_X) rollout status deployment -n namespace-lister namespace-lister-proxy-auth
	KUBECONFIG=$(KUBECONFIG_ATSA) \
	KONFLUX_ADDRESS=$(KONFLUX_ADDRESS) \
	E2E_USE_INSECURE_TLS=true \
		go test $(CWD)/... -v --godog.tags=wip --godog.concurrency=1

.PHONY: test
test: vet clean create-test-identity export-test-identity-kubeconfig
	$(KUBECTL_X) rollout status deployment -n namespace-lister namespace-lister
	$(KUBECTL_X) rollout status deployment -n namespace-lister namespace-lister-proxy-auth
	KUBECONFIG=$(KUBECONFIG_ATSA) \
	KONFLUX_ADDRESS=$(KONFLUX_ADDRESS) \
	E2E_USE_INSECURE_TLS=true \
		go test $(CWD)/... -v --godog.tags=~skip --godog.concurrency=1

