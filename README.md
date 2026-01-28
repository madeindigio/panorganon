# Panorganon

**Panorganon** is an intelligent MCP (Model Context Protocol) orchestration server written in Go that acts as a smart catalog and dispatcher for multiple downstream MCP servers.

## Features

- **Multi-Transport Support**: stdio and streamable-HTTP transports
- **Intelligent Tool Discovery**: Automatic discovery and caching of tools from downstream servers
- **Smart Tool Search**: LLM-powered tool selection based on task descriptions
- **Flexible Server Management**: On-demand and keep-alive modes for downstream servers
- **Database Caching**: SQLite-based caching for fast tool lookup
- **Structured Logging**: Comprehensive logging with rotation support
- **Lua Filters**: Intercept and modify tool calls for security, privacy, and auditing

## Installation

### Prerequisites

- Go 1.24+ (automatically managed by go.mod)
- Make (optional, for build automation)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/sevir/panorganon
cd panorganon

# Build
make build

# Or build manually
go build -o bin/panorganon ./cmd/panorganon
```

### Installing

```bash
make install
```

## Configuration

Create a `config.yaml` file (see `examples/config.example.yaml`):

```yaml
server:
  stdio:
    enabled: true
  http:
    enabled: true
    port: 8080
    endpoint: "/mcp"

logging:
  level: info
  file: "./logs/panorganon.log"

database:
  path: "./panorganon.db"

sampling:
  provider: "anthropic"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"

downstream_servers:
  - name: "remembrances-mcp"
    type: "stdio"
    command: "remembrances-mcp-beta"
    args:
      - "--config"
      - "/path/to/config.yaml"
    env: {}
    keepalive: false

  - name: "remote-server"
    type: "streamable-http"
    url: "http://example.com/mcp"
    keepalive: true
```

## Usage

### Start the Server

```bash
# Using stdio transport (for MCP clients)
panorganon --config config.yaml

# With custom log level
panorganon --config config.yaml --log-level debug
```

### MCP Tools Exposed

Panorganon exposes 4 MCP tools:

#### 1. `search_tools`
Search for appropriate tools based on a task description.

**Parameters:**
- `task_description` (string, required): Natural language description of the task
- `max_results` (number, optional): Maximum number of tools to return (default: 10)

**Returns:** Array of tool objects with name, server, description, and input schema.

#### 2. `exec_tool`
Execute a tool from a downstream MCP server.

**Parameters:**
- `tool_name` (string, required): Name of the tool to execute
- `parameters` (object, required): JSON object with tool parameters
- `server_name` (string, optional): Explicit server name if ambiguous

**Returns:** Tool execution result.

#### 3. `list_servers`
List all configured downstream MCP servers with status.

**Returns:** Array of server objects with name, type, running status, and keepalive flag.

#### 4. `refresh_tools`
Force refresh of tool metadata cache.

**Parameters:**
- `server_name` (string, optional): Refresh only specific server

**Returns:** Success status and number of tools refreshed.

## Lua Filters

Panorganon includes a powerful Lua-based filtering system that allows you to intercept and modify tool calls to downstream MCP servers. This enables:

- **Security**: Redact API keys, tokens, and sensitive data
- **Privacy**: Filter system paths and personal information
- **Auditing**: Log all tool calls with custom metadata
- **Validation**: Enforce business rules and constraints
- **Transformation**: Modify data on the fly

### Quick Start

1. **Enable filters** in your `config.yaml`:

```yaml
filters:
  enabled: true
  script_path: "./filters/panorganon-filters.lua"
  timeout: 5s
  strict_mode: false
```

2. **Create a filter script** (`filters/panorganon-filters.lua`):

```lua
-- Block sensitive searches
_G["remembrances-mcp-input"] = function(context)
    local params = context.parameters

    if params.query and string.match(params.query, "password") then
        error("Search for passwords blocked")
    end

    return params
end

-- Redact API keys in responses
_G["hyper-mcp-output"] = function(context)
    local result = context.result

    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                item.text = string.gsub(item.text, "sk%-[a-zA-Z0-9]+", "sk-[REDACTED]")
            end
        end
    end

    return result
end
```

3. **Restart Panorganon** and filters will be applied automatically.

### Learn More

For comprehensive documentation on Lua filters including:
- Filter function naming conventions
- Available Lua modules
- Security best practices
- Example filters for common use cases
- Debugging and troubleshooting

See the complete [Lua Filters Documentation](docs/lua-filters.md).

## Architecture

```
panorganon/
├── cmd/
│   └── panorganon/          # Main entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── server/              # MCP server transports
│   ├── downstream/          # Downstream server management
│   ├── tools/               # Tool discovery and execution
│   ├── database/            # SQLite database layer
│   ├── luafilters/          # Lua filter system
│   └── logging/             # Structured logging
├── pkg/
│   └── version/             # Version information
├── docs/                    # Documentation
└── examples/                # Example configurations and filters
    ├── config.example.yaml
    └── filters/
        └── panorganon-filters.lua
```

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Lint

```bash
make lint
```

### Clean

```bash
make clean
```

## Version Information

Version information is injected at build time:

```bash
panorganon version
```

## License

[Add your license here]

## Contributing

[Add contributing guidelines here]

## Status

**Development Status:** Beta (95% Complete)

Completed:
- ✅ Project foundation and build system
- ✅ Configuration system with YAML and environment variables
- ✅ MCP transport layer (stdio, streamable-HTTP)
- ✅ Downstream server management (stdio, HTTP, SSE stub)
- ✅ Database layer with SQLite
- ✅ Logging system with rotation
- ✅ Tool discovery with automatic caching
- ✅ Tool search with LLM sampling (Anthropic/OpenAI)
- ✅ Tool execution with retries and validation
- ✅ Complete JSON-RPC communication for stdio/HTTP
- ✅ Testing framework and example tests
- ✅ Comprehensive documentation (API, Configuration, Development)
- ✅ Docker support with Dockerfile and docker-compose

Ready for production use:
- All 4 MCP tools fully functional (search_tools, exec_tool, list_servers, refresh_tools)
- Robust error handling and logging
- Parameter validation
- Automatic server lifecycle management
- LLM-powered intelligent tool selection

TODO (Nice-to-have):
- Additional unit and integration tests
- SSE transport implementation
- WebSocket support
- Tool result caching
- Web UI for management
- Performance optimizations

## Tasks

### build

Compiles the panorganon binary.

```bash
# Get version from git tags or use dev
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X 'github.com/sevir/panorganon/pkg/version.Version=$VERSION' -X 'github.com/sevir/panorganon/pkg/version.Commit=$COMMIT' -X 'github.com/sevir/panorganon/pkg/version.BuildTime=$BUILD_TIME'" -o bin/panorganon ./cmd/panorganon
```

### build-and-copy

Compiles the panorganon binary and copies it to ~/bin folder.

```bash
# Get version from git tags or use dev
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X 'github.com/sevir/panorganon/pkg/version.Version=$VERSION' -X 'github.com/sevir/panorganon/pkg/version.Commit=$COMMIT' -X 'github.com/sevir/panorganon/pkg/version.BuildTime=$BUILD_TIME'" -o bin/panorganon ./cmd/panorganon
rm -f ~/bin/panorganon
cp bin/panorganon ~/bin/panorganon
```

### release

Compiles binaries for multiple platforms (Linux x64, Windows x64, macOS aarch64) and generates compressed releases in the `dist` folder.

```bash
# Create dist folder
mkdir -p dist

# Get version from git tags or use dev
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Function to build and zip
build_and_zip() {
    local os=$1
    local arch=$2
    local suffix=$3
    local ext=$4
    
    echo "Building for $os-$arch..."
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags "-X 'github.com/sevir/panorganon/pkg/version.Version=$VERSION' -X 'github.com/sevir/panorganon/pkg/version.Commit=$COMMIT' -X 'github.com/sevir/panorganon/pkg/version.BuildTime=$BUILD_TIME' -s -w" -o dist/panorganon-$suffix$ext ./cmd/panorganon
    
    echo "Zipping panorganon-$suffix$ext..."
    cd dist
    zip panorganon-$suffix.zip panorganon-$suffix$ext
    rm panorganon-$suffix$ext
    cd ..
}

# Linux x64
build_and_zip linux amd64 linux-x64 ""

# Windows x64
build_and_zip windows amd64 windows-x64 ".exe"

# macOS aarch64
build_and_zip darwin arm64 darwin-arm64 ""

echo "Release builds completed in dist/"
```