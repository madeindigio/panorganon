# Lua Filters in Panorganon

## Overview

Panorganon includes a powerful Lua-based filtering system that allows you to intercept and modify tool calls to downstream MCP servers. This enables security, privacy, auditing, and data transformation capabilities without modifying code.

## Features

- **Input Filters**: Modify parameters before sending to downstream servers
- **Output Filters**: Transform results before returning to clients
- **Security**: Redact sensitive data (API keys, tokens, credentials)
- **Privacy**: Filter system paths, IPs, and personal information
- **Auditing**: Log all tool calls with custom metadata
- **Validation**: Enforce business rules and constraints
- **Transformation**: Modify data structure and content on the fly

## Configuration

Add the following section to your `config.yaml`:

```yaml
filters:
  enabled: true                                  # Enable/disable filters
  script_path: "./filters/panorganon-filters.lua" # Path to Lua script
  timeout: 5s                                    # Max execution time per filter
  strict_mode: false                             # Fail on filter errors
  hot_reload: false                              # Enable hot-reloading (future)
  log_filtered_data: false                       # Log filtered data (debug)
```

### Configuration Options

- **enabled**: Global toggle for the filter system
- **script_path**: Absolute or relative path to your Lua filter script
- **timeout**: Maximum time a filter can run (prevents infinite loops)
- **strict_mode**: If `true`, filter errors abort the operation; if `false`, errors are logged and original data is used
- **log_filtered_data**: Enable detailed logging of filtered data (for debugging)

## Filter Function Naming Convention

Lua filter functions follow a strict naming convention:

### Input Filters
```lua
_G["<server-name>-input"] = function(context)
    -- Modify parameters before execution
    return context.parameters
end
```

### Output Filters
```lua
_G["<server-name>-output"] = function(context)
    -- Modify results after execution
    return context.result
end
```

### Global Fallback Filters
```lua
_G["global-input"] = function(context)
    -- Applied when no specific input filter exists
    return context.parameters
end

_G["global-output"] = function(context)
    -- Applied when no specific output filter exists
    return context.result
end
```

**Note**: Lua doesn't allow hyphens in function names, so we use `_G["function-name"]` syntax.

## Context Structure

### Input Filter Context

```lua
context = {
    server_name = "remembrances-mcp",  -- Downstream server name
    tool_name = "search_vectors",       -- Tool being called
    parameters = {                      -- Input parameters
        query = "search term",
        limit = 10
    },
    request_id = "uuid-string",        -- Unique request ID
    timestamp = 1706472000              -- Unix timestamp
}
```

### Output Filter Context

```lua
context = {
    server_name = "remembrances-mcp",  -- Downstream server name
    tool_name = "search_vectors",       -- Tool that was called
    result = {                          -- Execution result
        content = {
            {
                type = "text",
                text = "Result content"
            }
        },
        isError = false
    },
    duration = 150,                     -- Execution time in ms
    request_id = "uuid-string",        -- Unique request ID
    timestamp = 1706472000              -- Unix timestamp
}
```

## Available Lua Modules

The following modules are pre-loaded and available in filter scripts:

- **strings**: Advanced string manipulation
- **http**: HTTP client (use with caution)
- **time**: Time and date operations
- **re**: Regular expressions (Go regexp syntax)
- **yaml**: YAML parsing and generation
- **template**: Text templates
- **json**: JSON encoding/decoding
- **env**: Environment variable access

**Note**: The `sh` module is intentionally disabled for security reasons.

## Example Filters

### Example 1: Basic Security Filter

```lua
_G["remembrances-mcp-input"] = function(context)
    local params = context.parameters

    -- Block sensitive searches
    if params.query then
        if string.match(params.query, "password") or
           string.match(params.query, "api[_-]?key") then
            error("Search for sensitive information blocked")
        end
    end

    -- Truncate large content
    if params.content and #params.content > 10000 then
        params.content = string.sub(params.content, 1, 10000) .. "... [truncated]"
    end

    return params
end
```

### Example 2: Path Redaction Filter

```lua
_G["remembrances-mcp-output"] = function(context)
    local result = context.result

    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                -- Redact home directories
                item.text = string.gsub(item.text, "/home/[^/]+", "/home/[USER]")
                item.text = string.gsub(item.text, "/Users/[^/]+", "/Users/[USER]")
                item.text = string.gsub(item.text, "C:\\Users\\[^\\]+", "C:\\Users\\[USER]")
            end
        end
    end

    return result
end
```

### Example 3: API Key Redaction

```lua
_G["hyper-mcp-output"] = function(context)
    local result = context.result

    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                -- Redact OpenAI keys
                item.text = string.gsub(item.text, "sk%-[a-zA-Z0-9]+", "sk-[REDACTED]")
                -- Redact Anthropic keys
                item.text = string.gsub(item.text, "sk%-ant%-[a-zA-Z0-9%-]+", "sk-ant-[REDACTED]")
                -- Redact Bearer tokens
                item.text = string.gsub(item.text, "Bearer [a-zA-Z0-9_%-%.]+", "Bearer [REDACTED]")
            end
        end
    end

    return result
end
```

### Example 4: SSRF Prevention

```lua
_G["hyper-mcp-input"] = function(context)
    local params = context.parameters

    if params.url then
        -- Block localhost and private networks
        local blocked_patterns = {
            "localhost",
            "127%.0%.0%.1",
            "192%.168%.",
            "10%.%d+%.",
            "172%.1[6-9]%.",
            "172%.2[0-9]%.",
            "172%.3[0-1]%."
        }

        for _, pattern in ipairs(blocked_patterns) do
            if string.match(params.url, pattern) then
                error("URL to private network blocked: " .. pattern)
            end
        end
    end

    return params
end
```

### Example 5: Audit Logging

```lua
_G["global-input"] = function(context)
    local params = context.parameters

    -- Log all tool calls
    print(string.format("[AUDIT] %s.%s by request %s at %d",
        context.server_name,
        context.tool_name,
        context.request_id,
        context.timestamp))

    -- Add audit metadata
    params._audit = {
        timestamp = context.timestamp,
        request_id = context.request_id
    }

    return params
end
```

## Best Practices

### 1. Start with Global Filters

Begin with global filters for auditing and basic security, then add server-specific filters as needed:

```lua
-- Start with this
_G["global-input"] = function(context)
    -- Basic auditing for all servers
    print(string.format("[AUDIT] %s.%s", context.server_name, context.tool_name))
    return context.parameters
end

-- Add specific filters later
_G["sensitive-server-input"] = function(context)
    -- Specific validation for sensitive operations
    -- ...
    return context.parameters
end
```

### 2. Use Non-Strict Mode During Development

Set `strict_mode: false` while developing filters to avoid breaking your application:

```yaml
filters:
  enabled: true
  script_path: "./filters/dev-filters.lua"
  strict_mode: false  # Errors are logged but don't break operations
```

### 3. Test Filters Thoroughly

Always test your filters with real data before deploying to production:

```bash
# Run Panorganon with test configuration
panorganon --config config-test.yaml

# Check logs for filter execution
tail -f logs/panorganon.log | grep "Filter"
```

### 4. Keep Filters Simple and Fast

Filters run on every tool call, so keep them efficient:

- Avoid complex computations
- Don't make external HTTP calls unless necessary
- Use early returns when possible
- Cache expensive operations

### 5. Use Consistent Error Messages

Make debugging easier with clear error messages:

```lua
if not validate_email(params.email) then
    error("Invalid email format: " .. params.email)
end
```

### 6. Document Your Filters

Add comments explaining what each filter does:

```lua
-- Block searches for sensitive information
-- This prevents accidentally exposing passwords, API keys, etc.
_G["remembrances-mcp-input"] = function(context)
    -- ...
end
```

## Debugging

### Enable Debug Logging

```yaml
filters:
  enabled: true
  log_filtered_data: true  # Log all filtered data
```

```yaml
logging:
  level: debug  # Enable debug logs
```

### Check Filter Execution

```bash
# Watch filter logs in real-time
tail -f logs/panorganon.log | grep "luafilters"

# Check for errors
grep "Filter execution failed" logs/panorganon.log

# Check for timeouts
grep "Filter execution timeout" logs/panorganon.log
```

### Common Issues

**Problem**: Filter not executing

- Check that `filters.enabled: true` in config
- Verify `script_path` points to valid file
- Check function name matches server name exactly
- Look for syntax errors in Lua script

**Problem**: Filter times out

- Reduce complexity in filter logic
- Increase `timeout` in configuration
- Check for infinite loops in Lua code

**Problem**: Original data still appears despite filter

- Verify filter returns modified data
- Check that filter doesn't error (check logs)
- Ensure `strict_mode: false` isn't hiding errors

## Performance Considerations

### Filter Execution Overhead

- Each filter adds ~1-5ms of latency per call
- Timeout default is 5 seconds
- Filters run synchronously in the request path

### Optimization Tips

1. **Use specific filters instead of global**: Global filters run on every tool call
2. **Return early**: Exit as soon as possible
3. **Avoid regex when possible**: String matching is faster than regex
4. **Don't load modules unnecessarily**: Only use required modules
5. **Cache computed values**: Store expensive computations in Lua globals

## Security Considerations

### Sandboxing

The Lua environment is sandboxed:

- No `sh` module (shell execution disabled)
- No file write access (only read via `fs` module)
- No unrestricted network access
- Timeout enforcement prevents DoS

### Best Security Practices

1. **Never trust input**: Always validate parameters
2. **Whitelist, don't blacklist**: Allow known-good patterns rather than blocking known-bad
3. **Log security events**: Track blocked requests for analysis
4. **Review filters regularly**: Security requirements change over time
5. **Test with malicious input**: Try to break your own filters

## Migration from No Filters

To add filters to an existing Panorganon deployment:

1. **Start with filters disabled**:
   ```yaml
   filters:
     enabled: false
     script_path: "./filters/panorganon-filters.lua"
   ```

2. **Create minimal filter script** with global audit logging

3. **Enable filters in non-strict mode**:
   ```yaml
   filters:
     enabled: true
     strict_mode: false
   ```

4. **Monitor logs** for any issues

5. **Add specific filters** gradually for each server

6. **Enable strict mode** once confident:
   ```yaml
   filters:
     strict_mode: true
   ```

## Troubleshooting

### Filter Script Syntax Errors

```
ERROR: failed to load script: syntax error near '-'
```

**Solution**: Use `_G["function-name"]` syntax instead of `function function-name()`

### Module Not Available

```
ERROR: module 'fs' not found
```

**Solution**: Check that the module is in the available modules list. Not all gopher-lua-libs modules may be available.

### Memory Leaks

If you notice increasing memory usage:

1. Check for Lua table leaks in your filters
2. Avoid storing data in Lua globals
3. Return tables, don't mutate them in place
4. Consider reloading filters periodically (future feature)

## Support

For issues or questions:

- GitHub Issues: https://github.com/sevir/panorganon/issues
- Documentation: https://github.com/sevir/panorganon/docs

## License

Panorganon and its Lua filter system are licensed under the same license as the main project.
