
# Image URL to use all building/pushing image targets
IMG ?= alexdecb/l2sm-controller-manager:2.8.0
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

include .env
export DEV_IP


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker
KIND ?= kind

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

REPOSITORY=L2S-M

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test-e2e  # Run the e2e tests against a Kind k8s instance that is spun up.
test-e2e:
	go test ./test/e2e/ -v -ginkgo.v

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: install-requisites
install-requisites: add-cni
	$(KUBECTL) apply  -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.1/cert-manager.yaml; \
	$(KUBECTL) apply  -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml; \
	$(KUBECTL) wait --for=condition=Ready pods --all -A --timeout=300s; \



.PHONY: run
include .env
export $(shell sed 's/=.*//' .env)
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	echo "" > deployments/l2sm-deployment.yaml
	echo "---" >> deployments/l2sm-deployment.yaml  # Add a document separator before appending
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default >> deployments/l2sm-deployment.yaml


##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: webhook-certs
webhook-certs: ## generate self-signed cert and key for local webhook development
	mkdir -p /tmp/k8s-webhook-server/serving-certs
	sed  -e 's/{{IP_2}}/$(DEV_IP)/' ./config/dev/openssl.cnf > /tmp/openssl.cnf
	openssl req -x509 -newkey rsa:2048 -nodes -keyout /tmp/k8s-webhook-server/serving-certs/tls.key -out /tmp/k8s-webhook-server/serving-certs/tls.crt -days 365 -config /tmp/openssl.cnf -batch -subj '/CN=local-webhook'
	cat /tmp/k8s-webhook-server/serving-certs/tls.crt | base64 -w0 > /tmp/k8s-webhook-server/tls.b64






.PHONY: create-cluster
create-cluster:
	kind create cluster --config ./examples/quickstart/kind-cluster.yaml

.PHONY: clean
clean:
	$(KIND) delete clusters --all

.PHONY: add-cni
add-cni:
	@if [ ! -d "plugins/bin" ] || [ -z "$$(ls -A plugins/bin)" ]; then \
		mkdir -p plugins/bin; \
		wget -q https://github.com/containernetworking/plugins/releases/download/v1.6.0/cni-plugins-linux-amd64-v1.6.0.tgz; \
		tar -xf cni-plugins-linux-amd64-v1.6.0.tgz -C plugins/bin; \
		chmod +x plugins/bin/*; \
		rm cni-plugins-linux-amd64-v1.6.0.tgz; \
	fi
	@nodes="$$( $(KIND) get nodes -A)"; \
	if [ -z "$$nodes" ]; then \
		echo "No nodes found. Is Kind running?"; \
		exit 1; \
	fi; \
	for node in $$nodes; do \
		echo "Copying plugins to node: $$node"; \
		$(CONTAINER_TOOL) cp ./plugins/bin/. $$node:/opt/cni/bin; \
		if [ $$? -ne 0 ]; then \
			echo "Failed to copy plugins to $$node"; \
			exit 1; \
		fi; \
		$(CONTAINER_TOOL) exec $$node modprobe br_netfilter; \
		$(CONTAINER_TOOL) exec $$node sysctl -p /etc/sysctl.conf; \
	done; \
	$(KUBECTL) apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml; \
	if [ $$? -ne 0 ]; then \
		echo "Failed to install flannel in cluster"; \
		exit 1; \
	fi; \


.PHONY: copy-to-container
copy-to-container:
	@if [ -z "$(container)" ]; then \
		echo "Error: Please specify a container name using 'make copy-to-container container=<container_name>'"; \
		exit 1; \
	fi
	docker cp ./plugins/bin/. $(container):/opt/cni/bin

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name l2sm-test
	sudo rm -r ./plugins/
	
.PHONY: deploy-dev
deploy-dev: webhook-certs install manifests kustomize ## Deploy validating and mutating webhooks to the K8s cluster specified in ~/.kube/config.
	sed -i'' -e 's/caBundle: .*/caBundle: $(shell cat /tmp/k8s-webhook-server/tls.b64)/' ./config/dev/webhookcainjection_patch.yaml
	sed -i'' -e 's|url: .*|url: https://$(DEV_IP):9443/mutate-v1-pod|' ./config/dev/webhookcainjection_patch.yaml
	$(KUSTOMIZE) build config/dev | $(KUBECTL) apply -f -
	echo -e "CONTROLLER_IP=localhost\nCONTROLLER_PORT=30000" > .env
	
.PHONY: undeploy-dev
undeploy-dev: kustomize ## Undeploy validating and mutating webhooks from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/dev | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -


# Define file extensions for various formats
FILES := $(shell find . -type f \( -name "*.go" -o -name "*.json" -o -name "*.yaml" -o -name "*.yml" -o -name "*.md" \))

# Install the addlicense tool if not installed
.PHONY: install-tools
install-tools:
	GOBIN=$(LOCALBIN) go install github.com/google/addlicense@latest

# Add license headers to the files
.PHONY: add-license
add-license: install-tools
	@for file in $(FILES); do \
		$(LOCALBIN)/addlicense -f ./hack/LICENSE.txt -l apache "$${file}"; \
	done


##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

## Tool Versions
KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.17.2
ENVTEST_VERSION ?= latest
GOLANGCI_LINT_VERSION ?= v1.54.2

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})


# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef

