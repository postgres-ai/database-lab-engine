.PHONY: build build-linux build-lambda clean test fmt vet

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Build for current platform
build:
	go build $(LDFLAGS) -o rds-refresh .

# Build for Linux (for Docker/Lambda)
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o rds-refresh-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o rds-refresh-linux-arm64 .

# Build Lambda bootstrap binary
build-lambda:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bootstrap .
	zip rds-refresh-lambda.zip bootstrap
	rm bootstrap

# Clean build artifacts
clean:
	rm -f rds-refresh rds-refresh-linux-* bootstrap rds-refresh-lambda.zip

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
