IMAGE_REPO := "pringlewood"
VPP_VERSION := "20.01"

VERSION := $(shell git describe --always --tags --dirty)
COMMIT  := $(shell git rev-parse HEAD)
DATE    := $(shell date +'%Y-%m-%dT%H:%M%:z')

CNINFRA_AGENT := go.ligato.io/cn-infra/v2/agent
LDFLAGS := -s -w -X $(CNINFRA_AGENT).BuildVersion=$(VERSION) -X $(CNINFRA_AGENT).CommitHash=$(COMMIT) -X $(CNINFRA_AGENT).BuildDate=$(DATE)

CRDGEN_DEPS_DIR := "crdgen"

LINTER := $(shell golangci-lint --version 2>/dev/null)

cnf-crd:
	@echo "# building cnf-crd"
	cd cmd/cnf-crd && go build -v -ldflags "${LDFLAGS}" -tags="${GO_BUILD_TAGS}"

cnf-crd-image:
	./scripts/build-cnf-crd-image.sh $(IMAGE_REPO)

nsm-agent-vpp:
	@echo "# building nsm-agent-vpp"
	cd cmd/nsm-agent-vpp && go build -v -ldflags "${LDFLAGS}" -tags="${GO_BUILD_TAGS}"

nsm-agent-linux:
	@echo "# building nsm-agent-linux"
	cd cmd/nsm-agent-linux && go build -v -ldflags "${LDFLAGS}" -tags="${GO_BUILD_TAGS}"

nsm-agent-vpp-image:
	./scripts/build-vpp-agent-image.sh $(IMAGE_REPO) $(VPP_VERSION)

nsm-agent-linux-image:
	./scripts/build-linux-agent-image.sh $(IMAGE_REPO)

dep-install:
	go mod download

get-desc-adapter-generator:
	go install go.ligato.io/vpp-agent/v3/plugins/kvscheduler/descriptor-adapter

gen: get-desc-adapter-generator dep-install
	./scripts/gen-proto.sh
	cd ./plugins/nsmplugin && go generate .
	./plugins/crdplugin/scripts/update-codegen.sh ${CRDGEN_DEPS_DIR}

build-images: ## Build all images
	#make cnf-crd-image
	make nsm-agent-vpp-image
	make nsm-agent-linux-image

push-images: ## Push images to internal repo
	./scripts/push-images.sh $(IMAGE_REPO)

cleanup-images:
	./scripts/cleanup-images.sh $(IMAGE_REPO)

get-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.22.2
	golangci-lint --version

lint:
ifndef LINTER
	make get-linter
endif
	./scripts/static-analysis.sh
