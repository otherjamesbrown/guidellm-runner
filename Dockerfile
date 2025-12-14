# Dockerfile for guidellm-runner
# Includes Go runner + Python guidellm CLI

FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/guidellm-runner ./cmd/runner

# Final stage: Python runtime with guidellm
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install guidellm
RUN pip install --no-cache-dir guidellm

# Create non-root user
RUN useradd -m -s /bin/bash appuser

WORKDIR /app

# Copy Go binary from builder
COPY --from=builder /bin/guidellm-runner /app/guidellm-runner

# Create config and tmp directories
RUN mkdir -p /app/configs /tmp && chown -R appuser:appuser /app /tmp

USER appuser

# Expose metrics port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

ENTRYPOINT ["/app/guidellm-runner"]
CMD ["-config", "/app/configs/config.yaml"]
