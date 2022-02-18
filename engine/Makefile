.DEFAULT_GOAL = all

SERVER_BINARY = dblab-server
RUN_CI_BINARY = run-ci
CLI_BINARY = dblab
GOARCH = amd64

COMMIT?=$(shell git rev-parse HEAD)
BUILD_TIME?=$(shell date -u '+%Y%m%d-%H%M')
VERSION=$(shell git describe --tags 2>/dev/null || echo "${COMMIT}")

# Symlink into GOPATH
BUILD_DIR=${GOPATH}/${SERVER_BINARY}

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-s -w \
	-X gitlab.com/postgres-ai/database-lab/v3/version.version=${VERSION} \
	-X gitlab.com/postgres-ai/database-lab/v3/version.buildTime=${BUILD_TIME}"

# Go tooling command aliases
GOBUILD = GO111MODULE=on CGO_ENABLED=0 GOARCH=${GOARCH} go build ${LDFLAGS}
GOTEST = GO111MODULE=on go test -race

CLIENT_PLATFORMS=darwin linux freebsd windows
ARCHITECTURES=amd64

help: ## Display the help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-22s%s\n", $$1, $$2}'

all: clean build ## Build all binary components of the project

install-lint: ## Install the linter to $GOPATH/bin which is expected to be in $PATH
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.1

run-lint: ## Run linters
	golangci-lint run

lint: install-lint run-lint ## Install and run linters

build: ## Build binary files of all Database Lab components (Engine, CI Checker, CLI)
	${GOBUILD} -o bin/${SERVER_BINARY} ./cmd/database-lab/main.go
	${GOBUILD} -o bin/${RUN_CI_BINARY} ./cmd/runci/main.go
	${GOBUILD} -o bin/${CLI_BINARY} ./cmd/cli/main.go

build-ci-checker: ## Build the Database Lab CI Checker binary
	${GOBUILD} -o bin/${RUN_CI_BINARY} ./cmd/runci/main.go

build-client: ## Build Database Lab CLI binaries for all supported operating systems and platforms
	$(foreach GOOS, $(CLIENT_PLATFORMS),\
		$(foreach GOARCH, $(ARCHITECTURES), \
		$(shell \
			export GOOS=$(GOOS); \
			export GOARCH=$(GOARCH); \
			${GOBUILD} -o bin/cli/$(CLI_BINARY)-$(GOOS)-$(GOARCH) ./cmd/cli/main.go)))

build-image: ## Build Docker image with the locally compiled DLE binary
	docker build -t dblab_server:local  -f Dockerfile.dblab-server .

build-dle: build build-image ## Build Database Lab Engine binary and Docker image

test: ## Run unit tests
	${GOTEST} ./...

test-ci-integration: ## Run integration tests
	GO111MODULE=on go test -race  -tags=integration ./...

fmt: ## Format code
	go fmt $$(go list ./... | grep -v /vendor/)

clean: ## Remove compiled binaries from the local bin/ directory
	rm -f bin/*

.PHONY: help all build test run-lint install-lint lint fmt clean build-image build-dle build-ci-checker build-client build-ci-checker
