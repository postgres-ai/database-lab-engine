# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files first for better caching
COPY engine/go.mod engine/go.sum ./
RUN go mod download

# Copy source code
COPY engine/ ./

# Build the binary
ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /rds-refresh \
    ./cmd/rds-refresh

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 appuser

WORKDIR /app

COPY --from=builder /rds-refresh /usr/local/bin/rds-refresh

USER appuser

ENTRYPOINT ["/usr/local/bin/rds-refresh"]
CMD ["--help"]
