# Multi-stage Docker build for Gym Door Access Bridge
# Supports both amd64 and arm64 architectures

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    gcc \
    musl-dev \
    sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME

# Set build environment
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
ENV CGO_ENABLED=1

# Build the application
RUN go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o gym-door-bridge \
    ./cmd

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    tzdata \
    curl

# Create non-root user
RUN addgroup -g 1001 -S bridge && \
    adduser -u 1001 -S bridge -G bridge

# Create directories
RUN mkdir -p /app/data /app/config /app/logs && \
    chown -R bridge:bridge /app

# Copy binary from builder
COPY --from=builder /app/gym-door-bridge /app/gym-door-bridge

# Copy configuration template
COPY config.yaml.example /app/config/config.yaml.example

# Set working directory
WORKDIR /app

# Switch to non-root user
USER bridge

# Expose ports
EXPOSE 8080 8443

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/gym-door-bridge"]

# Default command
CMD ["--config", "/app/config/config.yaml", "--log-level", "info"]

# Labels
LABEL maintainer="Your Team <team@yourdomain.com>"
LABEL description="Gym Door Access Bridge - Connect door hardware to SaaS platform"
LABEL version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/yourdomain/gym-door-bridge"
LABEL org.opencontainers.image.documentation="https://docs.yourdomain.com/gym-door-bridge"
LABEL org.opencontainers.image.licenses="MIT"