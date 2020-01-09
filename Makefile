.DEFAULT_GOAL = all

BINARY = dblab
GOARCH = amd64

VERSION?=0.1
BUILD_TIME?=$(shell date -u '+%Y%m%d-%H%M')
COMMIT?=no #$(shell git rev-parse HEAD)
BRANCH?=no #$(shell git rev-parse --abbrev-ref HEAD)

# Symlink into GOPATH
BUILD_DIR=${GOPATH}/${BINARY}

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-s -w \
	-X main.version=${VERSION} \
	-X main.commit=${COMMIT} \
	-X main.branch=${BRANCH}\
	-X main.buildTime=${BUILD_TIME}"

# Go tooling command aliases
GOBUILD = GO111MODULE=on GOARCH=${GOARCH} go build ${LDFLAGS}
GOTEST = GO111MODULE=on go test
GORUN = GO111MODULE=on go run ${LDFLAGS}

# Build the project
all: clean build

# Install the linter to $GOPATH/bin which is expected to be in $PATH
install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.22.2

run-lint:
	golangci-lint run

lint: install-lint run-lint

build:
	 ${GOBUILD} -o bin/${BINARY} ./cmd/database-lab/

test:
	${GOTEST} ./pkg/...

fmt:
	go fmt $$(go list ./... | grep -v /vendor/)

clean:
	-rm -f bin/*

run:
	${GORUN} ./cmd/database-lab/*

.PHONY: all build test run-lint install-lint lint fmt clean run
