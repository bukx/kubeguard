# KubeGuard Makefile
IMG ?= bukx/kubeguard:latest
ENVTEST_K8S_VERSION = 1.30.0

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: fmt vet ## Run tests
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary
	go build -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet ## Run against the configured cluster
	go run cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image
	docker push ${IMG}

##@ Deployment

.PHONY: install
install: ## Install CRDs into the cluster
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the cluster
	kubectl delete -f config/crd/bases/

.PHONY: deploy
deploy: ## Deploy controller to the cluster
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/
	kubectl apply -f config/webhook/

.PHONY: undeploy
undeploy: ## Undeploy controller from the cluster
	kubectl delete -f config/manager/
	kubectl delete -f config/webhook/
	kubectl delete -f config/rbac/

.PHONY: samples
samples: ## Apply sample CRs
	kubectl apply -f config/samples/
