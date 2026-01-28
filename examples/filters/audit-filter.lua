-- audit-filter.lua
-- Comprehensive auditing and logging filters

-- ============================================
-- Global Audit Input Filter
-- ============================================

_G["global-input"] = function(context)
    local params = context.parameters

    -- Log all tool calls with detailed metadata
    local log_entry = string.format(
        "[AUDIT-INPUT] timestamp=%d request_id=%s server=%s tool=%s",
        context.timestamp,
        context.request_id,
        context.server_name,
        context.tool_name
    )

    print(log_entry)

    -- Log parameter summary (without sensitive data)
    local param_count = 0
    for k, v in pairs(params) do
        param_count = param_count + 1
    end
    print(string.format("[AUDIT-INPUT] parameters_count=%d", param_count))

    -- Log specific parameter types (without values)
    for key, value in pairs(params) do
        local value_type = type(value)
        print(string.format("[AUDIT-INPUT] param=%s type=%s", key, value_type))
    end

    -- Add audit metadata to parameters
    params._audit = {
        timestamp = context.timestamp,
        request_id = context.request_id,
        server_name = context.server_name,
        tool_name = context.tool_name,
        version = "1.0"
    }

    return params
end

-- ============================================
-- Global Audit Output Filter
-- ============================================

_G["global-output"] = function(context)
    local result = context.result

    -- Log execution completion
    local log_entry = string.format(
        "[AUDIT-OUTPUT] timestamp=%d request_id=%s server=%s tool=%s duration=%dms",
        context.timestamp,
        context.request_id,
        context.server_name,
        context.tool_name,
        context.duration
    )

    print(log_entry)

    -- Log result metadata
    if result.content then
        local content_count = #result.content
        print(string.format("[AUDIT-OUTPUT] content_items=%d", content_count))

        -- Log content types
        for i, item in ipairs(result.content) do
            print(string.format("[AUDIT-OUTPUT] item[%d] type=%s", i, item.type or "unknown"))
        end
    end

    -- Log if error occurred
    if result.isError then
        print(string.format("[AUDIT-OUTPUT] ERROR: Tool execution failed"))
    end

    -- Check for slow operations
    if context.duration > 5000 then
        print(string.format("[AUDIT-PERFORMANCE] SLOW OPERATION: %s.%s took %dms",
            context.server_name, context.tool_name, context.duration))
    end

    return result
end

-- ============================================
-- Specialized Audit Filters
-- ============================================

-- Audit filter for sensitive operations (e.g., remembrances-mcp)
_G["remembrances-mcp-input"] = function(context)
    local params = context.parameters

    -- Log data storage operations
    if context.tool_name == "add_vector" or
       context.tool_name == "save_fact" or
       context.tool_name == "save_event" then
        print(string.format("[AUDIT-DATA-WRITE] tool=%s request_id=%s",
            context.tool_name, context.request_id))

        -- Log user_id if present (important for compliance)
        if params.user_id then
            print(string.format("[AUDIT-DATA-WRITE] user_id=%s", params.user_id))
        end
    end

    -- Log data retrieval operations
    if context.tool_name == "search_vectors" or
       context.tool_name == "get_fact" or
       context.tool_name == "search_events" then
        print(string.format("[AUDIT-DATA-READ] tool=%s request_id=%s",
            context.tool_name, context.request_id))
    end

    -- Log data deletion operations (critical for compliance)
    if context.tool_name == "delete_vector" or
       context.tool_name == "delete_fact" then
        print(string.format("[AUDIT-DATA-DELETE] CRITICAL tool=%s request_id=%s",
            context.tool_name, context.request_id))
    end

    return params
end

-- Audit filter for external API calls (e.g., hyper-mcp)
_G["hyper-mcp-input"] = function(context)
    local params = context.parameters

    -- Log external URL access
    if params.url then
        -- Extract domain for logging (without full URL)
        local domain = string.match(params.url, "https?://([^/]+)")
        print(string.format("[AUDIT-EXTERNAL] domain=%s tool=%s request_id=%s",
            domain or "unknown", context.tool_name, context.request_id))
    end

    -- Log search operations
    if params.query then
        -- Log that a search was performed (without the query content)
        print(string.format("[AUDIT-SEARCH] tool=%s request_id=%s query_length=%d",
            context.tool_name, context.request_id, #params.query))
    end

    return params
end

-- ============================================
-- Compliance and Retention Helpers
-- ============================================

-- Generate compliance log entry
function generate_compliance_log(context, operation_type)
    return string.format(
        "[COMPLIANCE] timestamp=%d operation=%s server=%s tool=%s request_id=%s",
        context.timestamp,
        operation_type,
        context.server_name,
        context.tool_name,
        context.request_id
    )
end

-- Check if operation needs special compliance logging
function needs_compliance_logging(tool_name)
    local compliance_tools = {
        "delete_vector",
        "delete_fact",
        "update_vector",
        "save_event"
    }

    for _, tool in ipairs(compliance_tools) do
        if tool_name == tool then
            return true
        end
    end

    return false
end
