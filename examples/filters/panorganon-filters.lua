-- panorganon-filters.lua
-- Example Lua filters for different MCP servers in Panorganon
--
-- Filter Functions Naming Convention:
--   Input filters:  <server-name>-input(context)
--   Output filters: <server-name>-output(context)
--
-- Context Structure:
--   context.server_name  - Name of the MCP server
--   context.tool_name    - Name of the tool being called
--   context.parameters   - Input parameters (for input filters)
--   context.result       - Output result (for output filters)
--   context.request_id   - Unique request identifier
--   context.timestamp    - Unix timestamp
--   context.duration     - Execution time in ms (for output filters)

-- ============================================
-- Filters for remembrances-mcp
-- ============================================

function remembrances-mcp-input(context)
    local params = context.parameters

    -- Prevent searches for sensitive information
    if params.query then
        -- Block searches for passwords or keys
        if string.match(params.query, "password") or
           string.match(params.query, "api[_-]?key") then
            error("Search for sensitive information blocked")
        end
    end

    -- Limit size of stored content
    if params.content and #params.content > 10000 then
        params.content = string.sub(params.content, 1, 10000) .. "... [truncated by filter]"
    end

    -- Add audit metadata
    params._audit = {
        timestamp = os.time(),
        server = context.server_name,
        tool = context.tool_name,
        request_id = context.request_id
    }

    return params
end

function remembrances-mcp-output(context)
    local result = context.result

    -- Redact absolute system paths
    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                -- Redact paths like /home/user/...
                item.text = string.gsub(item.text, "/home/[^/]+", "/home/[USER]")
                item.text = string.gsub(item.text, "/Users/[^/]+", "/Users/[USER]")
                -- Redact Windows paths
                item.text = string.gsub(item.text, "C:\\Users\\[^\\]+", "C:\\Users\\[USER]")
            end
        end
    end

    return result
end

-- ============================================
-- Filters for hyper-mcp
-- ============================================

function hyper-mcp-input(context)
    local params = context.parameters

    -- Validate URLs to prevent SSRF
    if params.url then
        -- Block localhost and private IPs
        if string.match(params.url, "localhost") or
           string.match(params.url, "127%.0%.0%.1") or
           string.match(params.url, "192%.168%.") or
           string.match(params.url, "10%.%d+%.") or
           string.match(params.url, "172%.1[6-9]%.") or
           string.match(params.url, "172%.2[0-9]%.") or
           string.match(params.url, "172%.3[0-1]%.") then
            error("URL to private network blocked")
        end
    end

    return params
end

function hyper-mcp-output(context)
    local result = context.result

    -- Redact API keys in responses (OpenAI, Anthropic, etc.)
    if result.content then
        for i, item in ipairs(result.content) do
            if item.type == "text" and item.text then
                -- Redact OpenAI keys
                item.text = string.gsub(item.text, "sk%-[a-zA-Z0-9]+", "sk-[REDACTED]")
                -- Redact Anthropic keys
                item.text = string.gsub(item.text, "sk%-ant%-[a-zA-Z0-9%-]+", "sk-ant-[REDACTED]")
                -- Redact generic API keys patterns
                item.text = string.gsub(item.text, "api[_-]?key[%s]*[:=][%s]*['\"]?([a-zA-Z0-9_-]+)['\"]?", "api_key: [REDACTED]")
                -- Redact Bearer tokens
                item.text = string.gsub(item.text, "Bearer [a-zA-Z0-9_%-%.]+", "Bearer [REDACTED]")
            end
        end
    end

    return result
end

-- ============================================
-- Global Filters (Fallback)
-- ============================================

-- This function is called if no specific input filter exists for a server
function global-input(context)
    -- Log all tool calls for audit purposes
    print(string.format("[AUDIT] Tool: %s, Server: %s, RequestID: %s",
        context.tool_name, context.server_name, context.request_id))

    -- Return parameters unchanged
    return context.parameters
end

-- This function is called if no specific output filter exists for a server
function global-output(context)
    -- You can add global output processing here
    -- For example, add a watermark to all responses

    local result = context.result

    -- Add execution metadata if not present
    if result.content and context.duration then
        -- Optionally log slow operations
        if context.duration > 1000 then
            print(string.format("[PERF] Slow operation: %s.%s took %dms",
                context.server_name, context.tool_name, context.duration))
        end
    end

    return result
end

-- ============================================
-- Utility Functions
-- ============================================

-- Helper function to redact a specific field in a table
function redact_field(tbl, field_name)
    if tbl[field_name] then
        tbl[field_name] = "[REDACTED]"
    end
end

-- Helper function to remove a field from a table
function remove_field(tbl, field_name)
    tbl[field_name] = nil
end

-- Helper function to validate a field against a regex pattern
function validate_field(tbl, field_name, pattern)
    if not tbl[field_name] then
        return false
    end
    return string.match(tostring(tbl[field_name]), pattern) ~= nil
end

-- Helper function to add a field to a table
function add_field(tbl, field_name, value)
    tbl[field_name] = value
end

-- Example: More advanced filter with multiple checks
function example-advanced-input(context)
    local params = context.parameters

    -- Validate required fields
    if not params.user_id then
        error("user_id is required")
    end

    -- Sanitize input
    if params.query then
        -- Remove potentially dangerous characters
        params.query = string.gsub(params.query, "[<>\"']", "")

        -- Limit query length
        if #params.query > 500 then
            params.query = string.sub(params.query, 1, 500)
        end
    end

    -- Add request metadata
    params._metadata = {
        filtered_at = os.time(),
        filter_version = "1.0"
    }

    return params
end
