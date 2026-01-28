# Panorganon Configuration Guide

This document describes all configuration options for Panorganon.

## Table of Contents

- [Configuration File](#configuration-file)
- [Server Configuration](#server-configuration)
- [Logging Configuration](#logging-configuration)
- [Database Configuration](#database-configuration)
- [LLM Sampling Configuration](#llm-sampling-configuration)
- [Downstream Servers](#downstream-servers)
- [Environment Variables](#environment-variables)
- [Example Configurations](#example-configurations)

---

## Configuration File

Panorganon uses YAML format for configuration. Specify the config file with the `--config` flag:

```bash
panorganon --config config.yaml
```

### Minimal Configuration

```yaml
server:
  stdio:
    enabled: true

logging:
  level: info
  file: "./logs/panorganon.log"

database:
  path: "./panorganon.db"

sampling:
  provider: "anthropic"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"

downstream_servers: []
```

---

## Server Configuration

Configure MCP transport layer(s) that Panorganon will use to communicate with clients.

### Structure

```yaml
server:
  stdio:
    enabled: boolean
  http:
    enabled: boolean
    port: number
    endpoint: string
```

### stdio Transport

Use for direct process communication (most common for MCP clients).

**Parameters:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | true | Whether to enable stdio transport |

**Example:**

```yaml
server:
  stdio:
    enabled: true
```

### HTTP Transport

Use for HTTP-based MCP communication (streamable-http protocol).

**Parameters:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | false | Whether to enable HTTP transport |
| `port` | number | 8080 | Port to listen on |
| `endpoint` | string | "/mcp" | HTTP endpoint path |

**Example:**

```yaml
server:
  http:
    enabled: true
    port: 8080
    endpoint: "/mcp"
```

### Multiple Transports

You can enable both transports simultaneously:

```yaml
server:
  stdio:
    enabled: true
  http:
    enabled: true
    port: 8080
    endpoint: "/mcp"
```

---

## Logging Configuration

Configure structured logging with automatic rotation.

### Structure

```yaml
logging:
  level: string
  file: string
```

### Parameters

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | "info" | Log level: `debug`, `info`, `warn`, `error` |
| `file` | string | "./logs/panorganon.log" | Path to log file |

### Example

```yaml
logging:
  level: debug
  file: "/var/log/panorganon/panorganon.log"
```

### Log Rotation

Logs are automatically rotated using lumberjack with:
- Max size: 100 MB per file
- Max backups: 10 files
- Max age: 30 days
- Compression: enabled

### CLI Override

You can override logging configuration via command-line flags:

```bash
panorganon --config config.yaml --log-level debug --log-file /tmp/panorganon.log
```

---

## Database Configuration

Configure SQLite database for caching tool metadata.

### Structure

```yaml
database:
  path: string
```

### Parameters

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | "./panorganon.db" | Path to SQLite database file |

### Example

```yaml
database:
  path: "/var/lib/panorganon/data.db"
```

### Database Features

- **Schema**: Automatically created on first run
- **Mode**: WAL (Write-Ahead Logging) for better performance
- **Synchronous**: NORMAL mode for balanced performance/safety
- **Foreign Keys**: Enabled

### Database Schema

Two main tables:

**servers:**
- `id` (PRIMARY KEY)
- `name` (UNIQUE)
- `type`
- `config`
- `status`
- `last_seen`

**tools:**
- `id` (PRIMARY KEY)
- `server_id` (FOREIGN KEY)
- `name`
- `description`
- `input_schema`
- `last_updated`

---

## LLM Sampling Configuration

Configure which LLM provider to use for intelligent tool search.

### Structure

```yaml
sampling:
  provider: string
  api_key: string
  model: string
```

### Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | Yes | LLM provider: `anthropic` or `openai` |
| `api_key` | string | Yes | API key for the provider (supports env vars) |
| `model` | string | Yes | Model identifier to use |

### Anthropic Configuration

```yaml
sampling:
  provider: "anthropic"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"
```

**Supported Models:**
- `claude-3-5-sonnet-20241022` (recommended)
- `claude-3-opus-20240229`
- `claude-3-haiku-20240307`

### OpenAI Configuration

```yaml
sampling:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4-turbo-preview"
```

**Supported Models:**
- `gpt-4-turbo-preview`
- `gpt-4`
- `gpt-3.5-turbo`

### API Key Security

**Use environment variables** instead of hardcoding API keys:

```yaml
sampling:
  api_key: "${ANTHROPIC_API_KEY}"
```

Then set the environment variable:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

---

## Downstream Servers

Configure MCP servers that Panorganon will orchestrate.

### Structure

```yaml
downstream_servers:
  - name: string
    type: string
    # Type-specific fields...
```

### Common Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique server name |
| `type` | string | Yes | Server type: `stdio`, `streamable-http`, `sse` |
| `keepalive` | boolean | No | Keep server running (default: false) |

### stdio Servers

For local command-line MCP servers.

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | Yes | Executable command |
| `args` | array | No | Command arguments |
| `env` | object | No | Environment variables |

**Example:**

```yaml
downstream_servers:
  - name: "remembrances-mcp"
    type: "stdio"
    command: "remembrances-mcp-beta"
    args:
      - "--config"
      - "/path/to/config.yaml"
    env:
      DATABASE_URL: "sqlite:///data/db.sqlite"
    keepalive: false
```

### streamable-http Servers

For remote HTTP-based MCP servers.

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Server URL (with protocol) |

**Example:**

```yaml
downstream_servers:
  - name: "remote-tools"
    type: "streamable-http"
    url: "https://tools.example.com/mcp"
    keepalive: true
```

### SSE Servers

For Server-Sent Events based MCP servers.

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | SSE endpoint URL |

**Example:**

```yaml
downstream_servers:
  - name: "sse-tools"
    type: "sse"
    url: "https://sse.example.com/events"
    keepalive: true
```

**Note:** SSE support is currently a stub and not fully implemented.

### Keep-Alive Mode

Control server lifecycle:

- **keepalive: true**: Server starts at Panorganon startup and runs continuously
- **keepalive: false** (default): Server starts on-demand and stops after use

**When to use keepalive:**
- Slow-starting servers
- Frequently used tools
- Servers with initialization overhead
- Critical servers that should always be ready

**When to use on-demand:**
- Infrequently used servers
- Resource-intensive servers
- Many servers configured (to conserve resources)

---

## Environment Variables

### Supported Variables

Panorganon supports environment variable expansion in config files using `${VAR_NAME}` syntax.

**Example:**

```yaml
sampling:
  api_key: "${ANTHROPIC_API_KEY}"

downstream_servers:
  - name: "remote"
    type: "streamable-http"
    url: "${REMOTE_SERVER_URL}"
    env:
      DATABASE_URL: "${DB_CONN_STRING}"
```

### Common Environment Variables

```bash
# LLM API keys
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."

# Server URLs
export REMOTE_SERVER_URL="https://server.example.com"

# Database connections
export DB_CONN_STRING="postgresql://user:pass@host/db"
```

---

## Example Configurations

### Development Setup

Minimal configuration for local development:

```yaml
server:
  stdio:
    enabled: true

logging:
  level: debug
  file: "./logs/panorganon.log"

database:
  path: "./panorganon.db"

sampling:
  provider: "anthropic"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"

downstream_servers:
  - name: "local-tools"
    type: "stdio"
    command: "npx"
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "/tmp"
    keepalive: false
```

### Production Setup

Multi-server production configuration:

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
  file: "/var/log/panorganon/panorganon.log"

database:
  path: "/var/lib/panorganon/data.db"

sampling:
  provider: "anthropic"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"

downstream_servers:
  # Critical keepalive server
  - name: "core-tools"
    type: "stdio"
    command: "core-mcp-server"
    args: ["--production"]
    env:
      LOG_LEVEL: "info"
    keepalive: true

  # Remote HTTP server
  - name: "remote-services"
    type: "streamable-http"
    url: "https://mcp.internal.company.com/api"
    keepalive: true

  # On-demand tools
  - name: "filesystem-tools"
    type: "stdio"
    command: "filesystem-mcp"
    args: ["/data"]
    keepalive: false
```

### Multi-Environment Setup

Use environment-specific configs:

**config.dev.yaml:**
```yaml
sampling:
  model: "claude-3-haiku-20240307"  # Faster/cheaper for dev

logging:
  level: debug

downstream_servers:
  - name: "test-server"
    type: "stdio"
    command: "./test-mcp-server"
```

**config.prod.yaml:**
```yaml
sampling:
  model: "claude-3-5-sonnet-20241022"  # Best quality for prod

logging:
  level: info

downstream_servers:
  - name: "prod-server"
    type: "streamable-http"
    url: "https://prod-mcp.company.com"
```

Usage:
```bash
panorganon --config config.dev.yaml   # Development
panorganon --config config.prod.yaml  # Production
```

---

## Validation

Panorganon validates configuration on startup. Common validation errors:

### Missing Required Fields

```
Error: sampling.provider is required
```

**Fix:** Add the required field to your config.

### Invalid Values

```
Error: server.http.port must be between 1 and 65535
```

**Fix:** Use a valid port number.

### Unknown Server Type

```
Error: unknown server type: xyz
```

**Fix:** Use `stdio`, `streamable-http`, or `sse`.

### No Transports Enabled

```
Error: at least one transport (stdio or http) must be enabled
```

**Fix:** Enable at least one transport in the `server` section.

---

## Configuration Tips

1. **Start Simple**: Begin with stdio transport and one downstream server
2. **Use Environment Variables**: Never commit API keys to version control
3. **Enable Keepalive Selectively**: Only for frequently-used or slow-starting servers
4. **Monitor Logs**: Set appropriate log levels (debug for troubleshooting, info for production)
5. **Test Incrementally**: Add servers one at a time and test with `list_servers`
6. **Use Absolute Paths**: For database and log files in production
7. **Document Custom Settings**: Add comments to your config file

---

## Troubleshooting

### Server Won't Start

Check:
- Config file syntax (valid YAML)
- Required fields present
- Port not already in use (for HTTP)
- File permissions (logs, database)

### Downstream Server Issues

Check:
- Command/executable exists and is in PATH
- Correct arguments provided
- Environment variables set
- Server supports MCP protocol

### LLM Sampling Failures

Check:
- API key is valid
- Model name is correct
- Network connectivity
- API rate limits

### Database Errors

Check:
- Directory exists for database file
- Write permissions
- Disk space available
- No corruption (delete and restart if needed)

---

## Security Considerations

1. **API Keys**: Always use environment variables
2. **File Permissions**: Restrict config file access (chmod 600)
3. **Logs**: Be careful not to log sensitive data
4. **Network**: Use HTTPS for remote servers
5. **Database**: Protect database file (contains tool metadata)
6. **Validation**: Panorganon validates tool parameters before execution

---

## Migration Guide

### From v0.1.0 to Future Versions

No breaking changes expected. Future versions will be backward compatible where possible.

---

## Reference

See also:
- [API.md](API.md) - MCP tool documentation
- [DEVELOPMENT.md](DEVELOPMENT.md) - Development setup
- [README.md](../README.md) - Getting started

---

For questions or issues, visit: https://github.com/sevir/panorganon
