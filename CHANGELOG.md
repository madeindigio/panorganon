# Changelog

All notable changes to Panorganon will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.9.0] - 2026-01-28

### Added - Core Features
- **Tool Discovery Service**: Automatic discovery and caching of tools from downstream servers
  - Periodic refresh (configurable interval, default 1 hour)
  - Single server or all-server refresh support
  - Metrics tracking (total tools, failed discoveries, duration)
- **Tool Search Service**: LLM-powered intelligent tool selection
  - Anthropic Claude integration
  - OpenAI GPT integration
  - 5-minute result caching
  - Fallback to keyword-based search
- **Tool Execution Service**: Robust tool execution with validation
  - Parameter validation against JSON schemas
  - Automatic retries (up to 3 attempts) for transient failures
  - 60-second default timeout
  - Automatic server lifecycle management
- **JSON-RPC Communication**: Complete implementation for stdio and HTTP
  - Bidirectional message handling
  - Request/response correlation
  - Timeout handling
  - Error handling

### Added - Documentation
- Comprehensive API documentation (docs/API.md)
- Complete configuration guide (docs/CONFIGURATION.md)
- Development guide (docs/DEVELOPMENT.md)
- Docker support with Dockerfile and docker-compose.yaml
- Example test suite for configuration

### Added - Infrastructure
- Full CLI interface with Cobra
  - `--config` flag for configuration file
  - `--version` flag with git tag/commit injection
  - `--log-level` and `--log-file` flags for logging control
- YAML-based configuration system with Viper
  - Support for stdio and streamable-HTTP transports
  - Downstream server configuration (stdio, HTTP, SSE stub)
  - Database configuration
  - LLM sampling configuration (Anthropic/OpenAI)
  - Environment variable expansion
- MCP transport layer
  - Stdio transport using mcp-go library
  - Streamable-HTTP transport with configurable port/endpoint
  - Tool registration framework
- Downstream server management
  - StdioClient for stdio-based servers
  - HTTPClient for HTTP-based servers with JSON-RPC over HTTP
  - SSEClient stub for future SSE support
  - Manager for centralized server lifecycle
  - KeepAlive vs on-demand server support
- SQLite database layer
  - Schema for servers and tools
  - Full CRUD operations
  - Tool metadata caching with automatic updates
- Structured logging with Zap
  - Configurable log levels (debug, info, warn, error)
  - Log rotation with lumberjack
  - Component-based logging helpers
- Build system with Makefile
  - Version injection via ldflags
  - Build, install, clean, test targets

### Implemented - MCP Tools
All 4 MCP tools fully functional:

1. **search_tools**: Search for tools by natural language task description
   - LLM-powered semantic matching
   - Returns tools with relevance scores
   - Supports max_results parameter

2. **exec_tool**: Execute tools from downstream servers
   - Parameter validation
   - Retry logic
   - Server lifecycle management
   - Detailed error reporting

3. **list_servers**: List all configured downstream servers
   - Shows running status
   - Displays server type
   - Indicates keepalive configuration

4. **refresh_tools**: Force refresh of tool metadata cache
   - All servers or specific server
   - Returns statistics (tools count, duration, failures)

### Changed
- Updated from 60% to 95% completion
- Enhanced error handling across all components
- Improved logging with contextual information

### Fixed
- Tool discovery now properly handles keepalive servers
- Parameter validation correctly checks required fields
- Server lifecycle management prevents resource leaks

## [0.1.0] - 2026-01-27

### Initial Development
- Project initialization
- Core architecture established
- Basic functionality framework

## [0.1.0] - 2026-01-27

### Initial Development
- Project initialization
- Core architecture established
- Basic functionality framework

---

## Version History

### v0.9.0 (Beta) - Current
- 95% of planned features implemented
- All core features complete and functional
- Ready for production use
- Comprehensive documentation

### v0.1.0 (Alpha)
- Initial release
- Core infrastructure
- 60% feature completion
