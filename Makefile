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
GOBUILD = GO111MODULE=on GOARCH=${GOARCH} go build ${LDFLAGS}
GOTEST = GO111MODULE=on go test -race 
GORUN = GO111MODULE=on go run ${LDFLAGS}

CLIENT_PLATFORMS=darwin linux freebsd windows
ARCHITECTURES=amd64

# Build the project
all: clean build

# Install the linter to $GOPATH/bin which is expected to be in $PATH
install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.1

run-lint:
	golangci-lint run

lint: install-lint run-lint

build:
	${GOBUILD} -o bin/${SERVER_BINARY} ./cmd/database-lab/main.go
	${GOBUILD} -o bin/${RUN_CI_BINARY} ./cmd/runci/main.go
	${GOBUILD} -o bin/${CLI_BINARY} ./cmd/cli/main.go

build-ci-checker:
	${GOBUILD} -o bin/${RUN_CI_BINARY} ./cmd/runci/main.go

build-client:
	$(foreach GOOS, $(CLIENT_PLATFORMS),\
		$(foreach GOARCH, $(ARCHITECTURES), \
		$(shell \
			export GOOS=$(GOOS); \
			export GOARCH=$(GOARCH); \
			${GOBUILD} -o bin/cli/$(CLI_BINARY)-$(GOOS)-$(GOARCH) ./cmd/cli/main.go)))

test:
	${GOTEST} ./...

test-ci-integration:
	GO111MODULE=on go test -race  -tags=integration ./...

fmt:
	go fmt $$(go list ./... | grep -v /vendor/)

clean:
	rm -f bin/*

run:
	${GORUN} ./cmd/database-lab/*

.PHONY: all build test run-lint install-lint lint fmt clean run
