# Multi-stage build for Panorganon

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 panorganon && \
    adduser -D -u 1000 -G panorganon panorganon

# Create directories
RUN mkdir -p /app/logs /app/data && \
    chown -R panorganon:panorganon /app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/panorganon /usr/local/bin/panorganon

# Copy example config
COPY --from=builder /build/examples/config.example.yaml /app/config.yaml

# Switch to non-root user
USER panorganon

# Expose HTTP port (if using HTTP transport)
EXPOSE 8080

# Volume for persistent data
VOLUME ["/app/data", "/app/logs"]

# Set default config path
ENV CONFIG_FILE=/app/config.yaml

# Run panorganon
ENTRYPOINT ["panorganon"]
CMD ["--config", "/app/config.yaml"]
