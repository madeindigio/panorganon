# Panorganon Development Guide

Guide for developers who want to contribute to or build upon Panorganon.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [Building](#building)
- [Testing](#testing)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [Development Workflow](#development-workflow)

---

## Prerequisites

### Required Tools

- **Go**: Version 1.24 or higher
- **Make**: For build automation (optional)
- **Git**: For version control

### Optional Tools

- **Docker**: For containerized deployment
- **golangci-lint**: For code linting
- **goreleaser**: For release builds

### Installation

**Install Go:**
```bash
# On macOS
brew install go

# On Ubuntu/Debian
sudo apt-get install golang-go

# Or download from https://golang.org/dl/
```

**Verify Installation:**
```bash
go version  # Should be 1.24+
```

---

## Project Structure

```
panorganon/
├── cmd/
│   └── panorganon/          # Main application entry point
│       └── main.go
├── internal/                # Private application code
│   ├── config/             # Configuration management
│   │   └── config.go
│   ├── database/           # SQLite database layer
│   │   ├── client.go
│   │   ├── schema.go
│   │   └── queries.go
│   ├── downstream/         # Downstream server management
│   │   ├── types.go        # Interfaces
│   │   ├── manager.go      # Lifecycle manager
│   │   ├── stdio.go        # Stdio client
│   │   ├── http.go         # HTTP client
│   │   ├── sse.go          # SSE client (stub)
│   │   └── jsonrpc.go      # JSON-RPC communication
│   ├── logging/            # Structured logging
│   │   └── logger.go
│   ├── server/             # MCP server transports
│   │   ├── server.go       # Transport factory
│   │   ├── stdio.go        # Stdio transport
│   │   ├── http.go         # HTTP transport
│   │   └── handler.go      # Tool handlers
│   └── tools/              # Tool services
│       ├── discovery.go    # Tool discovery
│       ├── search.go       # LLM-powered search
│       ├── sampling.go     # LLM API integration
│       └── executor.go     # Tool execution
├── pkg/                    # Public libraries
│   └── version/            # Version information
│       └── version.go
├── docs/                   # Documentation
│   ├── API.md
│   ├── CONFIGURATION.md
│   └── DEVELOPMENT.md
├── examples/               # Example configurations
│   └── config.example.yaml
├── Makefile               # Build automation
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── README.md
├── TODO.md
└── CHANGELOG.md
```

### Package Organization

- **cmd/**: Application entry points
- **internal/**: Private packages (cannot be imported by other projects)
- **pkg/**: Public packages (can be reused)
- **docs/**: Documentation
- **examples/**: Example files

---

## Building

### Quick Build

```bash
# Using Make
make build

# Or with Go directly
go build -o bin/panorganon ./cmd/panorganon
```

### Development Build

```bash
# Build with race detector
go build -race -o bin/panorganon ./cmd/panorganon

# Build with debug symbols
go build -gcflags="all=-N -l" -o bin/panorganon ./cmd/panorganon
```

### Production Build

```bash
make build
```

This injects version information:
- Git tag or "dev"
- Git commit hash
- Build timestamp

### Install Locally

```bash
make install
# Installs to $GOPATH/bin/panorganon
```

### Clean Build Artifacts

```bash
make clean
```

---

## Testing

### Run Tests

```bash
# All tests
make test

# Or with Go
go test ./...

# With verbose output
go test -v ./...

# Specific package
go test ./internal/config

# With coverage
go test -cover ./...
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Run integration tests (when implemented)
go test -tags=integration ./...
```

---

## Architecture

### Core Components

#### 1. Configuration System (`internal/config`)

- Loads YAML configuration
- Validates settings
- Expands environment variables
- Uses Viper library

#### 2. Database Layer (`internal/database`)

- SQLite for tool metadata caching
- Stores servers and tools
- Schema auto-initialization
- CRUD operations

#### 3. Downstream Management (`internal/downstream`)

- **Manager**: Orchestrates server lifecycle
- **Clients**: Stdio, HTTP, SSE implementations
- **JSON-RPC**: Protocol communication
- **Lifecycle**: Start, stop, keep-alive

#### 4. Tool Services (`internal/tools`)

- **Discovery**: Finds and caches tools
- **Search**: LLM-powered tool selection
- **Executor**: Executes tools with retries
- **Sampling**: Anthropic/OpenAI integration

#### 5. MCP Server (`internal/server`)

- **Transports**: Stdio and HTTP
- **Handlers**: Implements 4 MCP tools
- Uses mcp-go library

#### 6. Logging (`internal/logging`)

- Structured logging with Zap
- Automatic rotation with Lumberjack
- Configurable levels

### Data Flow

```
MCP Client
    ↓
Panorganon (stdio/HTTP transport)
    ↓
Tool Handler (search_tools, exec_tool, etc.)
    ↓
Service Layer (Discovery, Search, Executor)
    ↓
Database (cache) + Downstream Manager
    ↓
Downstream Server (via JSON-RPC)
```

### Key Interfaces

**DownstreamClient:**
```go
type DownstreamClient interface {
    Start(ctx context.Context) error
    Stop() error
    IsRunning() bool
    CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
    ListTools(ctx context.Context) ([]mcp.Tool, error)
    GetName() string
    GetType() string
}
```

### Dependencies

Major dependencies (see go.mod):
- `github.com/mark3labs/mcp-go`: MCP protocol implementation
- `github.com/spf13/cobra`: CLI framework
- `github.com/spf13/viper`: Configuration management
- `go.uber.org/zap`: Structured logging
- `gopkg.in/natefinch/lumberjack.v2`: Log rotation
- `modernc.org/sqlite`: Pure-Go SQLite driver

---

## Contributing

### Code Style

Follow standard Go conventions:
- `gofmt` for formatting
- `golint` for linting
- Meaningful variable names
- Comments for exported functions

### Formatting

```bash
# Format all code
go fmt ./...

# Or
gofmt -s -w .
```

### Linting

```bash
# Install golangci-lint
brew install golangci-lint

# Run linter
golangci-lint run
```

### Commit Messages

Use conventional commits:
```
feat: add SSE client support
fix: resolve tool discovery timeout
docs: update configuration guide
test: add executor service tests
```

### Pull Requests

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Update documentation
6. Submit pull request

---

## Development Workflow

### 1. Setup

```bash
# Clone repository
git clone https://github.com/sevir/panorganon
cd panorganon

# Install dependencies
go mod download

# Build
make build
```

### 2. Running Locally

```bash
# Create config file
cp examples/config.example.yaml config.yaml

# Edit with your settings
$EDITOR config.yaml

# Run with stdio
./bin/panorganon --config config.yaml

# Or with debug logging
./bin/panorganon --config config.yaml --log-level debug
```

### 3. Testing Changes

```bash
# Run tests
make test

# Build and test
make build
./bin/panorganon version
```

### 4. Adding New Features

#### Example: Add a New Tool Handler

1. **Define tool in `internal/server/handler.go`:**
```go
myTool := mcp.NewTool("my_tool",
    mcp.WithDescription("My new tool"),
    mcp.WithString("param", mcp.Required()),
)

s.AddTool(myTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Implementation
})
```

2. **Add service if needed** in `internal/tools/`

3. **Update tests**

4. **Update documentation** in `docs/API.md`

#### Example: Add New Downstream Client Type

1. **Implement interface** in `internal/downstream/newtype.go`

2. **Register in factory** (`internal/downstream/manager.go`)

3. **Update config** validation

4. **Add tests**

5. **Document** in `docs/CONFIGURATION.md`

### 5. Debugging

```bash
# Run with debug logging
./bin/panorganon --config config.yaml --log-level debug

# Use delve debugger
dlv debug ./cmd/panorganon -- --config config.yaml

# Race detector
go run -race ./cmd/panorganon --config config.yaml
```

### 6. Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Memory profiling
go test -memprofile=mem.prof -bench=.

# View profile
go tool pprof cpu.prof
```

---

## Project Phases

Panorganon was developed in phases:

- ✅ **Phase 1**: Foundation (project structure, modules)
- ✅ **Phase 2**: Configuration system
- ✅ **Phase 3**: MCP transport layer
- ✅ **Phase 4**: Downstream server management
- ✅ **Phase 5**: Database layer
- ✅ **Phase 6**: Tool discovery
- ✅ **Phase 7**: Tool search with LLM
- ✅ **Phase 8**: Tool execution
- ✅ **Phase 9**: Logging system
- 🔄 **Phase 10**: Testing & Documentation (in progress)

Current status: **90% complete**

---

## Known Issues

See [TODO.md](../TODO.md) for:
- Missing features
- Known bugs
- Planned improvements
- Performance optimizations

---

## Release Process

### Version Numbering

Follows [Semantic Versioning](https://semver.org/):
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### Creating a Release

1. Update CHANGELOG.md
2. Tag version:
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin v0.2.0
   ```
3. Build with version:
   ```bash
   make build
   ```

---

## Resources

### Go Development

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Package Layout](https://github.com/golang-standards/project-layout)

### MCP Protocol

- [MCP Go Library](https://github.com/mark3labs/mcp-go)
- [MCP Specification](https://modelcontextprotocol.io)

### Tools

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [viper](https://github.com/spf13/viper) - Configuration
- [zap](https://github.com/uber-go/zap) - Logging

---

## Getting Help

- **Issues**: https://github.com/sevir/panorganon/issues
- **Discussions**: https://github.com/sevir/panorganon/discussions
- **Documentation**: See docs/ directory

---

## License

[Specify your license here]

---

Happy coding! 🚀
