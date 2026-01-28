.PHONY: build clean install test lint run

# Binary name
BINARY_NAME=panorganon

# Get version from git tags, fallback to dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -X 'github.com/sevir/panorganon/pkg/version.Version=$(VERSION)' \
           -X 'github.com/sevir/panorganon/pkg/version.Commit=$(COMMIT)' \
           -X 'github.com/sevir/panorganon/pkg/version.BuildTime=$(BUILD_TIME)'

# Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/panorganon

# Build for production (smaller binary)
build-prod:
	@echo "Building $(BINARY_NAME) $(VERSION) for production..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -o bin/$(BINARY_NAME) ./cmd/panorganon

# Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install -ldflags "$(LDFLAGS)" ./cmd/panorganon

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Run the application (requires config file)
run: build
	@./bin/$(BINARY_NAME) --config examples/config.example.yaml

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Development mode with hot reload (requires air)
dev:
	@air

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
