# Panorganon API Documentation

Panorganon exposes 4 MCP tools that provide intelligent orchestration of downstream MCP servers.

## Table of Contents

- [search_tools](#search_tools)
- [exec_tool](#exec_tool)
- [list_servers](#list_servers)
- [refresh_tools](#refresh_tools)

---

## search_tools

Search for appropriate tools based on a task description using LLM-powered semantic search.

### Description

Uses intelligent sampling with Large Language Models (Anthropic Claude or OpenAI GPT) to analyze your task description and recommend the most relevant tools from all available downstream servers. Falls back to keyword-based search if LLM sampling fails.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task_description` | string | Yes | Natural language description of the task you want to accomplish |
| `max_results` | number | No | Maximum number of tools to return (default: 10) |

### Returns

Array of tool objects with the following structure:

```json
[
  {
    "name": "tool_name",
    "server": "server_name",
    "description": "Tool description",
    "input_schema": {
      "type": "object",
      "properties": {...},
      "required": [...]
    },
    "score": 0.95
  }
]
```

### Response Fields

- `name` (string): The exact name of the tool
- `server` (string): The server that provides this tool
- `description` (string): What the tool does
- `input_schema` (object): JSON Schema describing the tool's parameters
- `score` (number): Relevance score from 0.0 to 1.0 (1.0 = most relevant)

### Example Usage

**Request:**
```json
{
  "task_description": "I need to search for documents about machine learning",
  "max_results": 5
}
```

**Response:**
```json
[
  {
    "name": "search_documents",
    "server": "docs-server",
    "description": "Search documents using semantic similarity",
    "input_schema": {
      "type": "object",
      "properties": {
        "query": {"type": "string"}
      },
      "required": ["query"]
    },
    "score": 0.95
  },
  {
    "name": "find_files",
    "server": "filesystem-server",
    "description": "Find files by name or content",
    "input_schema": {
      "type": "object",
      "properties": {
        "pattern": {"type": "string"}
      }
    },
    "score": 0.72
  }
]
```

### Features

- **LLM-Powered Selection**: Uses configured LLM (Anthropic/OpenAI) for intelligent tool matching
- **Caching**: Results are cached for 5 minutes to improve performance
- **Fallback**: Automatically falls back to keyword search if LLM fails
- **Multi-Server**: Searches across all configured downstream servers

---

## exec_tool

Execute a tool from a downstream MCP server.

### Description

Executes a specific tool with the provided parameters. This is the primary way to invoke tools discovered through `search_tools`. Handles server lifecycle (starting/stopping), parameter validation, retries, and timeouts automatically.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `tool_name` | string | Yes | Name of the tool to execute |
| `parameters` | object | Yes | JSON object with tool parameters matching the tool's input schema |
| `server_name` | string | No | Explicit server name if the tool exists on multiple servers |

### Returns

Returns the actual result from the downstream tool execution. The format depends on the specific tool being executed.

### Response on Success

The response is passed directly from the downstream tool. Common formats:

```json
{
  "content": [
    {
      "type": "text",
      "text": "Tool execution result..."
    }
  ]
}
```

### Response on Error

```json
{
  "error": "Error message describing what went wrong"
}
```

### Example Usage

**Request:**
```json
{
  "tool_name": "search_documents",
  "parameters": {
    "query": "machine learning algorithms",
    "limit": 10
  },
  "server_name": "docs-server"
}
```

**Response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 3 documents:\n1. Introduction to ML\n2. Deep Learning Basics\n3. Neural Networks"
    }
  ]
}
```

### Features

- **Automatic Server Management**: Starts servers on-demand and stops them if not keepalive
- **Parameter Validation**: Validates parameters against the tool's input schema
- **Retries**: Automatically retries transient failures up to 3 times
- **Timeout**: 60-second default timeout for tool execution
- **Error Handling**: Detailed error messages for troubleshooting

### Execution Flow

1. Looks up tool in database (optionally filtered by server)
2. Validates parameters against tool's JSON schema
3. Starts the downstream server (if not already running)
4. Executes the tool via JSON-RPC
5. Stops the server (if not keepalive)
6. Returns the result

---

## list_servers

List all configured downstream MCP servers with their status.

### Description

Returns information about all downstream servers configured in Panorganon, including their current status (running/stopped), type, and whether they are configured for keepalive mode.

### Parameters

None

### Returns

Array of server objects:

```json
[
  {
    "name": "server_name",
    "type": "stdio",
    "running": true,
    "keepalive": false
  }
]
```

### Response Fields

- `name` (string): Server name as configured
- `type` (string): Server type (`stdio`, `streamable-http`, or `sse`)
- `running` (boolean): Whether the server is currently running
- `keepalive` (boolean): Whether the server runs continuously

### Example Usage

**Request:**
```json
{}
```

**Response:**
```json
[
  {
    "name": "remembrances-mcp",
    "type": "stdio",
    "running": true,
    "keepalive": true
  },
  {
    "name": "remote-tools",
    "type": "streamable-http",
    "running": false,
    "keepalive": false
  }
]
```

### Use Cases

- Check server availability before using tools
- Monitor which servers are currently active
- Verify server configuration
- Debug connectivity issues

---

## refresh_tools

Force refresh of the tool metadata cache.

### Description

Triggers immediate discovery of tools from downstream servers and updates the database cache. Useful after adding new servers, updating existing servers, or when tool definitions change.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `server_name` | string | No | Refresh only a specific server (if omitted, refreshes all servers) |

### Returns

Success message with statistics:

```json
{
  "message": "Successfully refreshed tools from all servers",
  "total_tools": 42,
  "failed_servers": 0,
  "duration": "2.5s"
}
```

### Example Usage

**Refresh All Servers:**
```json
{}
```

**Response:**
```json
"Successfully refreshed tools from all servers\nTotal tools: 42\nFailed servers: 0\nDuration: 2.5s"
```

**Refresh Specific Server:**
```json
{
  "server_name": "remembrances-mcp"
}
```

**Response:**
```json
"Successfully refreshed 15 tools from server 'remembrances-mcp'"
```

### Features

- **Selective Refresh**: Can refresh all servers or just one
- **Automatic Discovery**: Connects to each server, lists tools, updates database
- **Server Lifecycle**: Starts/stops servers as needed (respects keepalive)
- **Error Handling**: Continues with other servers if one fails

### When to Use

- After modifying server configurations
- After downstream servers have been updated
- When tool definitions are out of sync
- Before critical operations that depend on up-to-date tool information

### Discovery Process

1. For each server (or the specified server):
   - Start the server if not running
   - Call the server's `tools/list` JSON-RPC method
   - Update server record in database
   - Delete old tool records
   - Save new tool records
   - Stop server if not keepalive
2. Report statistics (total tools, failed servers, duration)

---

## Error Handling

All tools return errors in a consistent format:

```json
{
  "error": "Error message"
}
```

Common error scenarios:

- **Service Not Initialized**: The required internal service hasn't been set up
- **Invalid Arguments**: Missing or malformed parameters
- **Server Not Found**: Specified server doesn't exist in configuration
- **Tool Not Found**: Specified tool doesn't exist in the database
- **Parameter Validation Failed**: Tool parameters don't match the schema
- **Execution Failed**: The downstream tool execution failed
- **Timeout**: Tool execution took too long

---

## Performance Considerations

### Caching

- **search_tools**: Results cached for 5 minutes (improves repeated searches)
- **Tool Metadata**: Stored in SQLite database for fast lookup
- **Discovery**: Runs automatically every 1 hour (configurable)

### Timeouts

- **search_tools**: 30 seconds for LLM API calls
- **exec_tool**: 60 seconds default for tool execution
- **refresh_tools**: 30 seconds per server for tool listing

### Retries

- **exec_tool**: Up to 3 retries for transient failures
- Retries only for: timeouts, connection issues, temporary failures
- Exponential backoff between retries

---

## Configuration

Tool behavior is configured via `config.yaml`:

```yaml
sampling:
  provider: "anthropic"  # or "openai"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"

database:
  path: "./panorganon.db"

downstream_servers:
  - name: "my-server"
    type: "stdio"
    command: "my-mcp-server"
    args: []
    keepalive: true
```

See [CONFIGURATION.md](CONFIGURATION.md) for complete configuration documentation.

---

## JSON-RPC Protocol

Panorganon uses JSON-RPC 2.0 for communication with downstream servers:

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {...}
}
```

### Supported Methods

- `tools/list`: List all available tools
- `tools/call`: Execute a specific tool

---

## Best Practices

1. **Use search_tools first**: Discover available tools before execution
2. **Check tool schemas**: Review `input_schema` before calling `exec_tool`
3. **Handle errors gracefully**: All tools can fail, implement retry logic
4. **Monitor server status**: Use `list_servers` to check availability
5. **Refresh periodically**: Call `refresh_tools` after server configuration changes
6. **Provide specific task descriptions**: Better descriptions = better search results
7. **Use server_name when needed**: Disambiguate tools with the same name

---

## Examples

### Complete Workflow

1. Search for tools:
```json
{
  "tool": "search_tools",
  "params": {
    "task_description": "find and analyze code files"
  }
}
```

2. Review results and select a tool

3. Execute the tool:
```json
{
  "tool": "exec_tool",
  "params": {
    "tool_name": "code_search",
    "parameters": {
      "pattern": "function.*main",
      "directory": "/src"
    }
  }
}
```

---

## Support

For issues, questions, or feature requests, please open an issue at:
https://github.com/sevir/panorganon/issues
