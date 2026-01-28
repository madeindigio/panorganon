# Panorganon - TODO List

## Phase 6: Tool Discovery (High Priority)

### Implementation Tasks
- [ ] Create `internal/tools/discovery.go` with DiscoveryService
- [ ] Implement `DiscoverAll()` method
  - Iterate all configured downstream servers
  - For each server: Start() -> ListTools() -> Store in DB -> Stop() (if not keepalive)
- [ ] Implement `DiscoverServer(serverName string)` for single server refresh
- [ ] Add periodic refresh with configurable interval (default 1 hour)
- [ ] Handle errors gracefully: log failures but continue with other servers
- [ ] Add metrics: total tools discovered, failed discoveries, last discovery time
- [ ] Update `refresh_tools` tool handler in `internal/server/handler.go`
- [ ] Initialize DiscoveryService in `cmd/panorganon/main.go`
- [ ] Trigger initial discovery on startup

### Expected Files
- `internal/tools/discovery.go`
- `internal/tools/metadata.go`

## Phase 7: Tool Search with MCP Sampling (High Priority)

### Implementation Tasks
- [ ] Create `internal/tools/search.go` with SearchService
- [ ] Implement `SearchTools(taskDescription string, maxResults int)` method
- [ ] Query database for all available tools
- [ ] Build MCP sampling request with system prompt for tool selection
- [ ] Integrate with configured LLM provider (Anthropic/OpenAI)
- [ ] Parse LLM response to extract recommended tool names and scores
- [ ] Fetch full tool metadata from database
- [ ] Implement fallback keyword-based search if sampling fails
- [ ] Add caching for sampling results (5 min TTL)
- [ ] Update `search_tools` tool handler in `internal/server/handler.go`
- [ ] Initialize SearchService in `cmd/panorganon/main.go`

### Expected Files
- `internal/tools/search.go`
- `internal/tools/sampling.go`

### External Dependencies
- Anthropic API client for sampling
- OpenAI API client for sampling (alternative)

## Phase 8: Tool Execution (High Priority)

### Implementation Tasks
- [ ] Create `internal/tools/executor.go` with ExecutorService
- [ ] Implement `ExecTool(toolName, params, serverName)` method
- [ ] Lookup tool in database (filter by serverName if provided)
- [ ] Validate params against inputSchema using JSON schema validator
- [ ] Call `Manager.GetOrStart(serverName)` to ensure server is running
- [ ] Marshal params to proper format
- [ ] Call downstream client's `CallTool()` via MCP protocol
- [ ] Wait for response with timeout (configurable, default 60s)
- [ ] Stop server if not keepalive
- [ ] Add execution logging: tool name, server, duration, success/failure
- [ ] Implement retry logic for transient failures (max 3 retries)
- [ ] Update `exec_tool` tool handler in `internal/server/handler.go`
- [ ] Initialize ExecutorService in `cmd/panorganon/main.go`

### Expected Files
- `internal/tools/executor.go`
- `internal/tools/validator.go`

### External Dependencies
- JSON schema validator library: `github.com/xeipuuv/gojsonschema`

## Phase 10: Testing & Documentation (Medium Priority)

### Testing Tasks
- [ ] Create unit tests for config parsing/validation
- [ ] Create unit tests for database queries
- [ ] Create unit tests for tool search logic
- [ ] Create unit tests for tool execution flow
- [ ] Create mock interfaces using gomock or testify/mock
- [ ] Create integration test: start server, call search_tools, call exec_tool
- [ ] Add test coverage reporting
- [ ] Add GitHub Actions workflow for CI/CD

### Documentation Tasks
- [ ] Complete `docs/API.md` with detailed tool documentation
- [ ] Complete `docs/CONFIGURATION.md` with full YAML schema
- [ ] Create `docs/DEVELOPMENT.md` with build and contribution guidelines
- [ ] Create example configurations in `examples/`:
  - `stdio-only.yaml`
  - `http-mixed.yaml`
  - `production.yaml`
- [ ] Create `Dockerfile` for containerized deployment
- [ ] Create `docker-compose.yaml` with example setup
- [ ] Add deployment guide for common platforms

### Expected Files
- `*_test.go` files for all packages
- `docs/API.md`
- `docs/CONFIGURATION.md`
- `docs/DEVELOPMENT.md`
- `examples/*.yaml`
- `Dockerfile`
- `docker-compose.yaml`
- `.github/workflows/ci.yaml`

## Critical TODOs (Must Fix)

### 1. Complete JSON-RPC Communication (Critical)
**Location**: `internal/downstream/stdio.go`, `internal/downstream/http.go`

Currently, `CallTool()` and `ListTools()` methods are stubs that return errors. Need to implement:

#### For stdio:
- [ ] Implement proper JSON-RPC 2.0 message framing over stdin/stdout
- [ ] Create request ID generation
- [ ] Implement bidirectional message handling
- [ ] Handle notifications vs requests
- [ ] Add timeout handling

#### For HTTP:
- [ ] Implement JSON-RPC 2.0 over HTTP
- [ ] Handle MCP session management
- [ ] Implement proper request/response correlation

**Priority**: CRITICAL - Tool execution won't work without this

### 2. Improve Error Handling
- [ ] Add custom error types
- [ ] Improve error messages with context
- [ ] Add error codes for different failure modes
- [ ] Implement circuit breaker for failing downstream servers

### 3. Add Monitoring & Metrics
- [ ] Add Prometheus metrics endpoint
- [ ] Track tool execution times
- [ ] Track downstream server health
- [ ] Add request tracing

### 4. Security Enhancements
- [ ] Add authentication for HTTP transport
- [ ] Implement rate limiting
- [ ] Add input validation for all tool parameters
- [ ] Sanitize log output (avoid logging secrets)

## Nice-to-Have Features (Low Priority)

- [ ] Add WebSocket transport support
- [ ] Implement tool result caching
- [ ] Add plugin system for custom tools
- [ ] Create web UI for server management
- [ ] Add OpenTelemetry tracing
- [ ] Implement tool composition (chaining tools)
- [ ] Add RBAC for tool access control
- [ ] Create CLI tool for testing tools (`panorganon-cli`)

## Known Issues

1. **Process monitoring incomplete**: KeepAlive server restart logic not implemented
2. **No connection pooling**: HTTP clients don't reuse connections
3. **No backpressure**: No rate limiting on tool execution
4. **Limited error recovery**: Failures aren't always handled gracefully

## Performance Optimizations

- [ ] Add connection pooling for HTTP clients
- [ ] Implement tool metadata caching in memory
- [ ] Add database query optimization with prepared statements
- [ ] Implement concurrent tool discovery
- [ ] Add request batching for multiple tool calls

## Documentation Improvements

- [ ] Add architecture diagrams
- [ ] Create video tutorial
- [ ] Add troubleshooting guide
- [ ] Create FAQ section
- [ ] Add performance tuning guide
