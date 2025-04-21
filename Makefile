IMG_TAG ?= dev
IMG_REGISTRY_NAMESPACE ?= ghcr.io/routesentry

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker



.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

#.PHONY: lint
#lint: golangci-lint ## Run golangci-lint linter
#	$(GOLANGCI_LINT) run
#
#.PHONY: lint-fix
#lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
#	$(GOLANGCI_LINT) run --fix
#
#.PHONY: lint-config
#lint-config: golangci-lint ## Verify golangci-lint linter configuration
#	$(GOLANGCI_LINT) config verify

##@ Build

.PHONY: build
build: fmt vet ## Build binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet
	go run ./cmd/main.go

PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
# If you wish to build the image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build-init
docker-build-init: ## Build docker image.
	$(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --target init -t "${IMG_REGISTRY_NAMESPACE}/routesentry-init:${IMG_TAG}" .

.PHONY: docker-build-sidecar
docker-build-sidecar: ## Build docker image.
	$(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --target sidecar -t "${IMG_REGISTRY_NAMESPACE}/routesentry-sidecar:${IMG_TAG}" .

# PLATFORMS defines the target platforms for the image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/myimage:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
.PHONY: docker-buildx-init
docker-buildx-init: ## Build and push docker image for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name routesentry-builder
	$(CONTAINER_TOOL) buildx use routesentry-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --target init --tag "${IMG_REGISTRY_NAMESPACE}/routesentry-init:${IMG_TAG}" -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm routesentry-builder
	rm Dockerfile.cross

.PHONY: docker-buildx-sidecar
docker-buildx-sidecar: ## Build and push docker image for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name routesentry-builder
	$(CONTAINER_TOOL) buildx use routesentry-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --target sidecar --tag "${IMG_REGISTRY_NAMESPACE}/routesentry-sidecar:${IMG_TAG}" -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm routesentry-builder
	rm Dockerfile.cross