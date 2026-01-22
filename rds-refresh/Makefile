.PHONY: build build-linux clean test fmt vet docker-build docker-push

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"
DOCKER_IMAGE ?= postgresai/rds-refresh

# Build for current platform
build:
	go build $(LDFLAGS) -o rds-refresh .

# Build for Linux (for Docker containers)
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o rds-refresh-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o rds-refresh-linux-arm64 .

# Clean build artifacts
clean:
	rm -f rds-refresh rds-refresh-linux-*

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run locally (requires config.yaml)
run:
	go run . -config config.yaml

# Run dry-run locally
dry-run:
	go run . -config config.yaml -dry-run

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Push Docker image
docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest
