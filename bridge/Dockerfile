# Go Bridge Dockerfile - Multi-stage build for production
# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy bridge source code
COPY bridge/ ./bridge/

# Build the bridge binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /build/bridge-server \
    ./bridge/*.go

# Runtime stage - minimal image
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 -S bridge && \
    adduser -u 1000 -S bridge -G bridge

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bridge-server /app/bridge-server

# Copy Helm chart (mounted at runtime or embedded)
# The chart path should be provided via BRIDGE_CHART_PATH env var

# Set ownership
RUN chown -R bridge:bridge /app

# Switch to non-root user
USER bridge

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Expose port
EXPOSE 8080

# Default environment variables (can be overridden)
ENV BRIDGE_LISTEN_ADDRESS=:8080
ENV BRIDGE_NAMESPACE=default
ENV BRIDGE_SYNC_INTERVAL=5m
ENV BRIDGE_SHUTDOWN_TIMEOUT=10s
ENV BRIDGE_HTTP_CLIENT_TIMEOUT=5s
ENV BRIDGE_OPERATION_TIMEOUT=10m
ENV BRIDGE_HEALTH_PATH=/healthz

# Run the bridge
ENTRYPOINT ["/app/bridge-server"]