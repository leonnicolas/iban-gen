export GO111MODULE=on
.PHONY: build migrate fmt lint lint-openapi mock vendor push container clean container-name container-latest push-latest manifest manfest-latest manifest-annotate manifest manfest-latest manifest-annotate test


OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
ALL_ARCH := amd64 arm arm64
DOCKER_ARCH := "amd64" "arm v7" "arm64 v8"
BIN_DIR := bin
BINS := $(BIN_DIR)/$(OS)/$(ARCH)/iban-gen
PROJECT := iban-gen
PKG := github.com/leonnicolas/$(PROJECT)

REGISTRY ?= index.docker.io
IMAGE ?= leonnicolas/$(PROJECT)
FULLY_QUALIFIED_IMAGE := $(REGISTRY)/$(IMAGE)
TAG := $(shell git describe --abbrev=0 --tags HEAD 2>/dev/null)
COMMIT := $(shell git rev-parse HEAD)
VERSION := $(COMMIT)
ifneq ($(TAG),)
    ifeq ($(COMMIT), $(shell git rev-list -n1 $(TAG)))
        VERSION := $(TAG)
    endif
endif
DIRTY := $(shell test -z "$$(git diff --shortstat 2>/dev/null)" || echo -dirty)
VERSION := $(VERSION)$(DIRTY)
LD_FLAGS := -ldflags "-s -w -X $(PKG)/version.Version=$(VERSION)"
SRC := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
BANK_DATA := $(shell find . -type f -wholename "./data/*")
GO_FILES ?= $$(find . -name '*.go' -not -path './vendor/*')
GO_PKGS ?= $$(go list ./... | grep -v "$(PKG)/vendor")

GOLINT_BINARY := $(BIN_DIR)/golint
OAPI_CODEGEN_BINARY := $(BIN_DIR)/oapi-codegen

BUILD_IMAGE ?= golang:1.18.0
BASE_IMAGE ?= scratch
CONTAINERIZE_BUILD ?= true
BUILD_SUFIX :=
ifeq ($(CONTAINERIZE_BUILD), true)
	BUILD_PREFIX := docker run --rm \
	    -u $$(id -u):$$(id -g) \
	    -v $$(pwd):/$(PROJECT) \
	    -w /$(PROJECT) \
	    $(BUILD_IMAGE) \
	    /bin/sh -c '
	BUILD_SUFIX := '
endif

build: $(BINS)

$(BINS): $(SRC) go.mod cmd/iban-gen/data
	@mkdir -p $(BIN_DIR)/$(word 2,$(subst /, ,$@))/$(word 3,$(subst /, ,$@))
	@echo "building: $@"
	@$(BUILD_PREFIX) \
	        GOARCH=$(word 3,$(subst /, ,$@)) \
	        GOOS=$(word 2,$(subst /, ,$@)) \
	        GOCACHE=$$(pwd)/.cache \
		CGO_ENABLED=0 \
		go build -mod=vendor -o $@ \
		    $(LD_FLAGS) \
		    ./cmd/$(@F) \
	$(BUILD_SUFIX)

$(BIN_DIR):
	mkdir -p $@

cmd/iban-gen/data: $(BANK_DATA)
	rm -r $@ || true
	cp -r $(<D) $@

container-latest-%:
	@$(MAKE) --no-print-directory ARCH=$* container-latest

container-%:
	@$(MAKE) --no-print-directory ARCH=$* container

push-latest-%:
	@$(MAKE) --no-print-directory ARCH=$* push-latest

push-%:
	@$(MAKE) --no-print-directory ARCH=$* push

all-build: $(addprefix build-$(OS)-, $(ALL_ARCH))

all-container: $(addprefix container-, $(ALL_ARCH))

all-push: $(addprefix push-, $(ALL_ARCH))

all-container-latest: $(addprefix container-latest-, $(ALL_ARCH))

all-push-latest: $(addprefix push-latest-, $(ALL_ARCH))

container: .container-$(ARCH)-$(VERSION) container-name
.container-$(ARCH)-$(VERSION): bin/linux/$(ARCH)/iban-gen Dockerfile
	@i=0; for a in $(ALL_ARCH); do [ "$$a" = $(ARCH) ] && break; i=$$((i+1)); done; \
	ia=""; iv=""; \
	j=0; for a in $(DOCKER_ARCH); do \
	    [ "$$i" -eq "$$j" ] && ia=$$(echo "$$a" | awk '{print $$1}') && iv=$$(echo "$$a" | awk '{print $$2}') && break; j=$$((j+1)); \
	done; \
		docker build -t $(IMAGE):$(ARCH)-$(VERSION) --build-arg FROM=$(BASE_IMAGE) --build-arg GOARCH=$(ARCH) .
	@docker images -q $(IMAGE):$(ARCH)-$(VERSION) > $@

container-latest: .container-$(ARCH)-$(VERSION)
	@docker tag $(IMAGE):$(ARCH)-$(VERSION) $(FULLY_QUALIFIED_IMAGE):$(ARCH)-latest
	@echo "container: $(IMAGE):$(ARCH)-latest"

container-name:
	@echo "container: $(IMAGE):$(ARCH)-$(VERSION)"

manifest: .manifest-$(VERSION) manifest-name
.manifest-$(VERSION): Dockerfile $(addprefix push-, $(ALL_ARCH))
	@docker manifest create --amend $(FULLY_QUALIFIED_IMAGE):$(VERSION) $(addsuffix -$(VERSION), $(addprefix $(FULLY_QUALIFIED_IMAGE):, $(ALL_ARCH)))
	@$(MAKE) --no-print-directory manifest-annotate-$(VERSION)
	@docker manifest push $(FULLY_QUALIFIED_IMAGE):$(VERSION) > $@

manifest-latest: Dockerfile $(addprefix push-latest-, $(ALL_ARCH))
	@docker manifest rm $(FULLY_QUALIFIED_IMAGE):latest || echo no old manifest
	@docker manifest create --amend $(FULLY_QUALIFIED_IMAGE):latest $(addsuffix -latest, $(addprefix $(FULLY_QUALIFIED_IMAGE):, $(ALL_ARCH)))
	@$(MAKE) --no-print-directory manifest-annotate-latest
	@docker manifest push $(FULLY_QUALIFIED_IMAGE):latest
	@echo "manifest: $(IMAGE):latest"

manifest-annotate: manifest-annotate-$(VERSION)

manifest-annotate-%:
	@i=0; \
	for a in $(ALL_ARCH); do \
	    annotate=; \
	    j=0; for da in $(DOCKER_ARCH); do \
		if [ "$$j" -eq "$$i" ] && [ -n "$$da" ]; then \
		    annotate="docker manifest annotate $(FULLY_QUALIFIED_IMAGE):$* $(FULLY_QUALIFIED_IMAGE):$$a-$* --os linux --arch"; \
		    k=0; for ea in $$da; do \
			[ "$$k" = 0 ] && annotate="$$annotate $$ea"; \
			[ "$$k" != 0 ] && annotate="$$annotate --variant $$ea"; \
			k=$$((k+1)); \
		    done; \
		    $$annotate; \
		fi; \
		j=$$((j+1)); \
	    done; \
	    i=$$((i+1)); \
	done

manifest-name:
	@echo "manifest: $(IMAGE):$(VERSION)"

push: .push-$(ARCH)-$(VERSION) push-name
.push-$(ARCH)-$(VERSION): .container-$(ARCH)-$(VERSION)
ifneq ($(REGISTRY),index.docker.io)
	@docker tag $(IMAGE):$(ARCH)-$(VERSION) $(FULLY_QUALIFIED_IMAGE):$(ARCH)-$(VERSION)
endif
	@docker push $(FULLY_QUALIFIED_IMAGE):$(ARCH)-$(VERSION)
	@docker images -q $(IMAGE):$(ARCH)-$(VERSION) > $@

push-latest: container-latest
	@docker push $(FULLY_QUALIFIED_IMAGE):$(ARCH)-latest
	@echo "pushed: $(IMAGE):$(ARCH)-latest"

push-name:
	@echo "pushed: $(IMAGE):$(ARCH)-$(VERSION)"

api/v1/v1.go: api/v1/v1.yaml $(OAPI_CODEGEN_BINARY)
	$(OAPI_CODEGEN_BINARY) -generate types,client,chi-server,spec -package v1 -o $@ $<

fmt:
	@echo $(GO_PKGS)
	gofmt -w -s $(GO_FILES)

lint: $(GOLINT_BINARY) lint-openapi
	@echo 'go vet $(GO_PKGS)'
	@vet_res=$$(GO111MODULE=on go vet -mod=vendor $(GO_PKGS) 2>&1); if [ -n "$$vet_res" ]; then \
		echo ""; \
		echo "Go vet found issues. Please check the reported issues"; \
		echo "and fix them if necessary before submitting the code for review:"; \
		echo "$$vet_res"; \
		exit 1; \
	fi
	@echo '$(GOLINT_BINARY) $(GO_PKGS)'
	@lint_res=$$($(GOLINT_BINARY) $(GO_PKGS)); if [ -n "$$lint_res" ]; then \
		echo ""; \
		echo "Golint found style issues. Please check the reported issues"; \
		echo "and fix them if necessary before submitting the code for review:"; \
		echo "$$lint_res"; \
		exit 1; \
	fi
	@echo 'gofmt -d -s $(GO_FILES)'
	@fmt_res=$$(gofmt -d -s $(GO_FILES)); if [ -n "$$fmt_res" ]; then \
		echo ""; \
		echo "Gofmt found style issues. Please check the reported issues"; \
		echo "and fix them if necessary before submitting the code for review:"; \
		echo "$$fmt_res"; \
		exit 1; \
	fi

lint-openapi:
	docker run --rm -v $$(pwd):/var/lib/iban-gen stoplight/spectral:5.9 lint /var/lib/iban-gen/api/v1/v1.yaml

test: cmd/iban-gen/data
	go test ./...

vendor:
	go mod tidy
	go mod vendor

mock:
	docker run --init --rm -v $$(pwd):/var/lib/iban-gen -p 4010:4010 stoplight/prism:4 mock -h 0.0.0.0 /var/lib/iban-gen/api/v1/v1.yaml

$(OAPI_CODEGEN_BINARY): | $(BIN_DIR)
	go build -mod=vendor -o $@ github.com/deepmap/oapi-codegen/cmd/oapi-codegen

$(GOLINT_BINARY):
	go build -mod=vendor -o $@ golang.org/x/lint/golint

clean: container-clean bin-clean
	rm -rf .cache

container-clean:
	rm -rf .container-* .manifest-* .push-*

bin-clean:
	rm -rf bin
