CWD ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR := $(CWD)/../..
OUT_DIR := $(CWD)/out
KIND_CLUSTER_NAME := namespace-lister-dumb-proxy
KUBECONFIG_ATSA := $(OUT_DIR)/namespace-lister-acceptance-tests-user.kcfg
KUBECONFIG_ADMIN := $(OUT_DIR)/namespace-lister-admin.kcfg
KONFLUX_ADDRESS := https://localhost:10443

include ../../make/*.mk

export KUBECONFIG=$(KUBECONFIG_ADMIN)

.PHONY: prepare
prepare: image-build kind-create kind-load-image deploy-test-infra deploy-namespace-lister deploy-test-proxy
	@:

.PHONY: deploy-test-proxy
deploy-test-proxy:
	$(KUBECTL) apply -k $(CWD)/config/proxy/

.PHONY: deploy-namespace-lister
deploy-namespace-lister: $(OUT_DIR)
	-rm -r $(OUT_DIR)/config
	mkdir -p $(OUT_DIR)/config
	( \
		cd $(OUT_DIR)/config && \
			kustomize init && \
			kustomize edit add base "../../../../../config" && \
			kustomize edit set namespace namespace-lister && \
			kustomize edit set image "namespace-lister:latest=$(IMG)" && \
			cp "$(ROOT_DIR)/../config/patches/with_cachenamespacelabelselector.yaml" . && \
			kustomize edit add patch \
				--path "with_cachenamespacelabelselector.yaml" \
				--group "apps" \
				--kind "Deployment" \
				--name "namespace-lister" \
				--version "v1" && \
			kustomize build | $(KUBECTL) apply -f - ; \
	)

.PHONY: wip
wip: vet clean create-test-identity export-test-identity-kubeconfig
	$(KUBECTL) rollout status deployment -n namespace-lister namespace-lister
	$(KUBECTL) rollout status deployment -n namespace-lister namespace-lister-proxy
	KUBECONFIG=$(KUBECONFIG_ATSA) \
	KONFLUX_ADDRESS=$(KONFLUX_ADDRESS) \
	E2E_USE_INSECURE_TLS=true \
		go test $(CWD)/... -v --godog.tags=wip --godog.concurrency=1

.PHONY: test
test: vet clean create-test-identity export-test-identity-kubeconfig
	$(KUBECTL) rollout status deployment -n namespace-lister namespace-lister
	$(KUBECTL) rollout status deployment -n namespace-lister namespace-lister-proxy
	KUBECONFIG=$(KUBECONFIG_ATSA) \
	KONFLUX_ADDRESS=$(KONFLUX_ADDRESS) \
	E2E_USE_INSECURE_TLS=true \
		go test $(CWD)/... -v --godog.tags=~skip --godog.concurrency=1
