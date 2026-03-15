.DEFAULT_GOAL = all

SERVER_BINARY = dblab-server
RUN_CI_BINARY = run-ci
CLI_BINARY = dblab
RDS_REFRESH_BINARY = rds-refresh
GOARCH = amd64

PWD= $(shell pwd)

COMMIT?=$(shell git rev-parse HEAD)
BUILD_TIME?=$(shell date -u '+%Y%m%d-%H%M')
VERSION=$(shell git describe --tags --match "v*" 2>/dev/null || echo "${COMMIT}")

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
ARCHITECTURES=amd64 arm64

help: ## Display the help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-22s%s\n", $$1, $$2}'

all: clean build ## Build all binary components of the project

install-lint: ## Install the linter to $GOPATH/bin which is expected to be in $PATH
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.5.0

run-lint: ## Run linters
	golangci-lint run

lint: install-lint run-lint ## Install and run linters

build: ## Build binary files of all Database Lab components (Engine, CI Checker, CLI, RDS Refresh)
	${GOBUILD} -o bin/${SERVER_BINARY} ./cmd/database-lab/main.go
	${GOBUILD} -o bin/${RUN_CI_BINARY} ./cmd/runci/main.go
	${GOBUILD} -o bin/${CLI_BINARY} ./cmd/cli/main.go
	${GOBUILD} -o bin/${RDS_REFRESH_BINARY} ./cmd/rds-refresh/main.go

build-debug: ## Build the Database Lab Server binary for debugging
	${GOBUILD} -ldflags "-X gitlab.com/postgres-ai/database-lab/v3/version.version=${VERSION} \
	-X gitlab.com/postgres-ai/database-lab/v3/version.buildTime=${BUILD_TIME}" \
	-gcflags="all=-N -l" -o bin/${SERVER_BINARY}-debug ./cmd/database-lab/main.go

build-ci-checker: ## Build the Database Lab CI Checker binary
	${GOBUILD} -o bin/${RUN_CI_BINARY} ./cmd/runci/main.go

build-rds-refresh: ## Build the RDS Refresh binary
	${GOBUILD} -o bin/${RDS_REFRESH_BINARY} ./cmd/rds-refresh/main.go

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

test-race: ## Run unit tests with race detector (explicit target for CI quality gate)
	${GOTEST} -count=1 -timeout=10m ./...

test-ci-integration: ## Run integration tests
	GO111MODULE=on go test -race  -tags=integration ./...

test-bench: ## Run benchmark tests
	GO111MODULE=on go test -bench=. -benchmem -count=3 -timeout=30m ./... 2>&1 | tee bench-results.txt

coverage-report: ## Run tests with coverage and report result
	@GO111MODULE=on go test -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | tail -1
	@TOTAL=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $${TOTAL}%"

vulnerability-scan: ## Scan dependencies for known vulnerabilities
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

quality: fmt run-lint test coverage-report ## Run all quality checks (format, lint, test, coverage)

fmt: ## Format code
	go fmt $$(go list ./... | grep -v /vendor/)

clean: ## Remove compiled binaries from the local bin/ directory
	rm -f bin/*

run-dle: build-dle
	docker run \
	--rm \
	--volume /tmp:/tmp \
	--volume /var/run/docker.sock:/var/run/docker.sock \
	--volume /var/lib/dblab:/var/lib/dblab/:rshared \
	--volume /var/lib/dblab/dblab_pool/dump:/var/lib/dblab/dblab_pool/dump \
	--volume /sys/kernel/debug:/sys/kernel/debug:rw \
	--volume /lib/modules:/lib/modules:ro \
    --volume $(PWD)/configs:/home/dblab/configs:ro \
    --volume $(PWD)/meta:/home/dblab/meta \
	--volume /proc:/host_proc:ro \
	--env DOCKER_API_VERSION=1.39 \
	-p "2345:2345" \
	dblab_server:local

.PHONY: help all build test test-race test-bench test-ci-integration run-lint install-lint lint fmt clean build-image build-dle build-ci-checker build-client build-rds-refresh run-dle coverage-report vulnerability-scan quality
