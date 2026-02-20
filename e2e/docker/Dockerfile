# Mau E2E Test Image
# Multi-stage build for optimized image size

# Stage 1: Builder
FROM golang:1.23-alpine AS builder

# Set GOTOOLCHAIN to allow using newer Go versions
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build Mau binary
RUN CGO_ENABLED=0 GOOS=linux go build -o mau ./cmd/mau/mau.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    gnupg \
    expect \
    && rm -rf /var/cache/apk/*

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mau /usr/local/bin/mau

# Create directories for Mau data
RUN mkdir -p /data/.mau

# Set environment variables (can be overridden at runtime)
ENV MAU_DATA_DIR=/data/.mau \
    MAU_LOG_LEVEL=info \
    MAU_HTTP_PORT=8080 \
    MAU_P2P_PORT=9090

# Expose ports
# HTTP server port
EXPOSE 8080
# P2P communication port
EXPOSE 9090

# Set working directory to data directory
WORKDIR /data

# Entrypoint script
COPY e2e/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["serve"]
