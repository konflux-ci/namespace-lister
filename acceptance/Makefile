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

.PHONY: patch-coredns
patch-coredns:
	kubectl apply -f ./dependencies/coredns/configmap.yaml

.PHONY: deploy-test-infra
deploy-test-infra: deploy-cert-manager deploy-test-ingress deploy-idp
	@:

.PHONY: deploy-test-ingress
deploy-test-ingress:
	kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml
	for i in {1..5}; do kubectl get -n ingress-nginx pod --selector 'app.kubernetes.io/component=controller' && break || sleep 2; done
	kubectl rollout status deployment --namespace ingress-nginx \
		--selector=app.kubernetes.io/component=controller \
		--timeout=90s

.PHONY: deploy-cert-manager
deploy-cert-manager: $(TMPDIR)
	kubectl apply -k ./dependencies/cert-manager/
	( \
		docker exec namespace-lister-control-plane cat /etc/kubernetes/pki/ca.crt > $(TMP_DIR)/ca.crt; \
		docker exec namespace-lister-control-plane cat /etc/kubernetes/pki/ca.key > $(TMP_DIR)/ca.key ; \
		kubectl create secret tls kube-root-ca \
			--namespace cert-manager \
			--cert=$(TMP_DIR)/ca.crt \
			--key=$(TMP_DIR)/ca.key \
			-o yaml --dry-run=client | kubectl apply -f -; \
	)
	for i in {1..5}; do kubectl get -n cert-manager deployment --selector 'app.kubernetes.io/instance=cert-manager' && break || sleep 2; done
	kubectl rollout status deployment -n cert-manager \
		--selector 'app.kubernetes.io/instance=cert-manager' \
		--timeout=300s 
	kubectl apply -k ./dependencies/cluster-issuer/

.PHONY: build-user1-kubeconfig
build-user1-kubeconfig:
	@kubectl rollout status -n keycloak statefulset keycloak --timeout=90s 1>&2
	@( \
		tkn=$$( \
			curl -f -s -k -L -X POST 'https://idp.namespacelister.localtest.me/idp/realms/redhat-external/protocol/openid-connect/token' \
        -H 'Content-Type: application/x-www-form-urlencoded' \
        --data-urlencode 'client_id=cloud-services' \
        --data-urlencode 'client_secret=client-secret' \
        --data-urlencode 'grant_type=password' \
        --data-urlencode 'scope=openid' \
        --data-urlencode 'username=user1' \
        --data-urlencode 'password=password' | jq '.access_token' -r ); \
		kind get kubeconfig --name namespace-lister | \
			yq '.users[0].user={"token":"'$${tkn}'"}' | \
			yq 'del(.clusters[0].cluster.certificate-authority-data)' \
	)

.PHONY: deploy-idp
deploy-idp:
	kubectl apply -k ./dependencies/keycloak/crd/
	kubectl apply -k ./dependencies/keycloak/deployment/
	for i in {1..10}; do kubectl get -n keycloak deployment keycloak-operator && break || sleep 2; done
	kubectl rollout status --namespace keycloak \
		deployment keycloak-operator \
		--timeout=300s
	for i in {1..10}; do kubectl get -n keycloak statefulset keycloak && break || sleep 2; done
	kubectl rollout status --namespace keycloak \
		statefulset keycloak \
		--timeout=90s

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
	# kubectl rollout status deployment -n keycloak -l 'app=sso'
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
TMP_DIR := $(ROOT_DIR)/tmp

GO ?= go

GOLANG_CI ?= $(GO) run -modfile $(shell dirname $(ROOT_DIR))/hack/tools/golang-ci/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint

## Local Folders
$(LOCALBIN):
	mkdir $(LOCALBIN)
$(OUTDIR):
	@mkdir $(OUTDIR)
$(TMP_DIR):
	@mkdir $(TMP_DIR)

.PHONY: lint
lint: ## Run go linter.
	$(GOLANG_CI) run ./...
