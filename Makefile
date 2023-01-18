############################################################
# Copyright 2022 Cisco Systems, Inc.
# See LICENSE.md for license information
############################################################

# git and build info
export BUILD_NUMBER ?= 0.0.0-0
export BUILD_HOST := $(shell hostname)
export GIT_HASH := $(shell git rev-parse --short --verify HEAD)
export GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
export GIT_OUTPUT := $(shell git status --porcelain)
export GIT_DIRTY  := $(if ${strip ${GIT_OUTPUT}},true,false)
export GIT_TIMESTAMP := $(shell git show -s --format=%ct)
export BUILD_TIMESTAMP := $(shell date +%s)

export VERSION_PKG_PATH := github.com/cisco-open/fsoc/cmd/version
export VERSION_INFO := \
-X ${VERSION_PKG_PATH}.defVersion=${BUILD_NUMBER} \
-X ${VERSION_PKG_PATH}.defGitHash=${GIT_HASH} \
-X ${VERSION_PKG_PATH}.defGitBranch=${GIT_BRANCH} \
-X ${VERSION_PKG_PATH}.defBuildHost=${BUILD_HOST} \
-X ${VERSION_PKG_PATH}.defGitDirty=${GIT_DIRTY} \
-X ${VERSION_PKG_PATH}.defGitTimestamp=${GIT_TIMESTAMP} \
-X ${VERSION_PKG_PATH}.defBuildTimestamp=${BUILD_TIMESTAMP}

DEV_BUILD_FLAGS := -ldflags='${VERSION_INFO} -X ${VERSION_PKG_PATH}.defIsDev=true'
PROD_BUILD_FLAGS := -ldflags='${VERSION_INFO} -X ${VERSION_PKG_PATH}.defIsDev=false'

GO           := go
SCRIPT_DIR ?= $(shell pwd)
GOTEST_OPT := -v -p 1 -race -timeout 60s

# choose files for formatting and other maintenance
GOFILES=$(shell find . -name '*.go' ! -name '*mock.go')

export CGO_ENABLED ?= 0
CURRENT_PATH := $(PATH)
TEST_REPORTS_DIR = ./build/reports

export PATH := bin:${PATH}
export GOBIN = $(SCRIPT_DIR)/bin
export PATH = $(GOBIN):$(CURRENT_PATH)

GORELEASER ?= $(GOBIN)/goreleaser
GOLINT ?= $(GOBIN)/golangci-lint
IMPI ?= $(GOBIN)/impi
GOCOVMERGE ?= $(GOBIN)/gocovmerge
GOACC ?= $(GOBIN)/go-acc
GOCOBERTURA ?= $(GOBIN)/gocover-cobertura

help:
	@grep -Eh '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


dev-build: ## Build the project
	@echo "Building ./fsoc ..."
	${GO} build -a ${DEV_BUILD_FLAGS}

dev-test: ## Test the project locally
	${GO} test

format:
	@echo "formatting code..."
	@goimports -w -local github.com/cisco-open/fsoc ${GOFILES}

vet:  ## Run go vet
	@echo "vetting code..."
	${GO} vet ./...

mod-update: ## Download all dependencies
	@echo "installing dependencies..."
	${GO} mod download

tidy: ## Tidy
	@echo "tidy..."
	${GO} mod tidy

lint: install-tools ## Linting go source code
	@echo "linting code..."
	${GOLINT} run ./...

go-impi: install-tools
	@$(IMPI) --local github.com/cisco-open/fsoc --scheme stdThirdPartyLocal ./...

#test:
#	${GO} test $(GOTEST_OPT) ./...

docs-generate: ## Run generate
	@echo "WIP"

.PHONY: install-tools
install-tools:
	${GO} install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.0
	${GO} install github.com/goreleaser/goreleaser@v1.12.3
	${GO} install golang.org/x/tools/cmd/goimports@v0.1.12
	${GO} install github.com/pavius/impi/cmd/impi@v0.0.3
	${GO} install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	${GO} install github.com/ory/go-acc@v0.2.8
	${GO} install github.com/t-yuki/gocover-cobertura@v0.0.0-20180217150009-aaee18c8195c

pre-commit: install-tools format go-impi lint vet tidy mod-update ## check all pre-req before committing
	@echo "pre commit checks completed"

build: install-tools mod-update
	@echo "Building binaries for all supported platforms in builds/"
	@${GORELEASER} release --snapshot --rm-dist --skip-publish

test-with-cover: install-tools
	$(GOACC) ./...
	mkdir -p $(TEST_REPORTS_DIR) && $(GOCOBERTURA) < coverage.txt > ./build/reports/coverage.xml

test: install-tools lint vet test-with-cover
