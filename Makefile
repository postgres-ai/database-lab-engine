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

# Build the project
all: clean dep vet main

dep:
	go get -v -d -t ./...

main:
	GOARCH=${GOARCH} go build ${LDFLAGS} -o bin/${BINARY} ./src/

test:
	go test ./src/

vet:
	go vet ./src/...

fmt:
	go fmt $$(go list ./... | grep -v /vendor/)

clean:
	-rm -f bin/*

run:
	go run ${LDFLAGS} ./src/*

.PHONY: all dep main test vet fmt clean run
