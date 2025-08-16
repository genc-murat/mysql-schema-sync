# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') \
              -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
              -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
              -X main.GoVersion=$(go version | cut -d' ' -f3) \
              -w -s" \
    -a -installsuffix cgo \
    -o mysql-schema-sync .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/mysql-schema-sync .

# Copy configuration examples
COPY --from=builder /app/examples ./examples

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose no ports (CLI application)

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./mysql-schema-sync version || exit 1

# Set entrypoint
ENTRYPOINT ["./mysql-schema-sync"]

# Default command (show help)
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="MySQL Schema Sync" \
      org.opencontainers.image.description="CLI tool for synchronizing MySQL database schemas" \
      org.opencontainers.image.vendor="MySQL Schema Sync Contributors" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/your-org/mysql-schema-sync" \
      org.opencontainers.image.documentation="https://github.com/your-org/mysql-schema-sync/blob/main/README.md"