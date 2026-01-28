-- transformation-filter.lua
-- Data transformation and enrichment filters

-- ============================================
-- Input Transformation Filters
-- ============================================

_G["remembrances-mcp-input"] = function(context)
    local params = context.parameters

    -- Normalize user_id format
    if params.user_id then
        -- Convert to lowercase and trim
        params.user_id = string.lower(params.user_id)
        params.user_id = string.gsub(params.user_id, "^%s*(.-)%s*$", "%1")
    end

    -- Normalize query strings
    if params.query then
        -- Trim whitespace
        params.query = string.gsub(params.query, "^%s*(.-)%s*$", "%1")

        -- Collapse multiple spaces
        params.query = string.gsub(params.query, "%s+", " ")
    end

    -- Add default values for optional parameters
    if params.limit == nil then
        params.limit = 10  -- Default limit
    end

    -- Clamp limit to reasonable range
    if params.limit and type(params.limit) == "number" then
        if params.limit > 100 then
            params.limit = 100  -- Max limit
        elseif params.limit < 1 then
            params.limit = 1  -- Min limit
        end
    end

    -- Add metadata
    params._meta = {
        processed_at = os.time(),
        filter_version = "1.0"
    }

    return params
end

_G["hyper-mcp-input"] = function(context)
    local params = context.parameters

    -- Ensure HTTPS for URLs
    if params.url and string.match(params.url, "^http://") then
        params.url = string.gsub(params.url, "^http://", "https://")
        print(string.format("[TRANSFORM] Upgraded HTTP to HTTPS for URL"))
    end

    -- Add user agent if not present
    if not params.user_agent then
        params.user_agent = "Panorganon/1.0"
    end

    -- Set default timeout
    if not params.timeout then
        params.timeout = 30  -- 30 seconds default
    end

    return params
end

-- ============================================
-- Output Transformation Filters
-- ============================================

_G["remembrances-mcp-output"] = function(context)
    local result = context.result

    if not result.content then
        return result
    end

    -- Transform text content
    for i, item in ipairs(result.content) do
        if item.type == "text" and item.text then
            -- Normalize line endings
            item.text = string.gsub(item.text, "\r\n", "\n")

            -- Remove excessive newlines
            item.text = string.gsub(item.text, "\n\n\n+", "\n\n")

            -- Trim trailing whitespace from lines
            item.text = string.gsub(item.text, "[ \t]+\n", "\n")
        end
    end

    -- Add processing metadata
    table.insert(result.content, {
        type = "text",
        text = string.format("\n[Processed by Panorganon at %d]", os.time())
    })

    return result
end

_G["hyper-mcp-output"] = function(context)
    local result = context.result

    if not result.content then
        return result
    end

    -- Transform and enrich content
    for i, item in ipairs(result.content) do
        if item.type == "text" and item.text then
            -- Remove tracking parameters from URLs
            item.text = remove_tracking_params(item.text)

            -- Sanitize HTML if present
            item.text = sanitize_html(item.text)

            -- Convert markdown links to a consistent format
            item.text = normalize_markdown_links(item.text)
        end
    end

    return result
end

-- ============================================
-- Global Transformation Filters
-- ============================================

_G["global-input"] = function(context)
    local params = context.parameters

    -- Add common metadata to all requests
    params._context = {
        server = context.server_name,
        tool = context.tool_name,
        request_id = context.request_id,
        timestamp = context.timestamp
    }

    -- Validate and sanitize string inputs
    for key, value in pairs(params) do
        if type(value) == "string" then
            -- Remove null bytes
            params[key] = string.gsub(value, "%z", "")

            -- Remove control characters except newline and tab
            params[key] = string.gsub(params[key], "[\001-\008\011-\012\014-\031\127]", "")
        end
    end

    return params
end

_G["global-output"] = function(context)
    local result = context.result

    -- Add execution metadata
    if not result._meta then
        result._meta = {}
    end

    result._meta.duration = context.duration
    result._meta.server = context.server_name
    result._meta.tool = context.tool_name
    result._meta.timestamp = context.timestamp

    return result
end

-- ============================================
-- Helper Functions
-- ============================================

-- Remove tracking parameters from URLs
function remove_tracking_params(text)
    -- Common tracking parameters
    local tracking_params = {
        "utm_source",
        "utm_medium",
        "utm_campaign",
        "utm_term",
        "utm_content",
        "fbclid",
        "gclid",
        "msclkid"
    }

    for _, param in ipairs(tracking_params) do
        -- Remove ?param=value or &param=value
        text = string.gsub(text, "%?" .. param .. "=[^&%s]+&", "?")
        text = string.gsub(text, "&" .. param .. "=[^&%s]+", "")
        text = string.gsub(text, "%?" .. param .. "=[^&%s]+", "")
    end

    -- Clean up trailing ? or &
    text = string.gsub(text, "[?&](%s)", "%1")
    text = string.gsub(text, "[?&]$", "")

    return text
end

-- Basic HTML sanitization
function sanitize_html(text)
    -- Remove script tags
    text = string.gsub(text, "<script[^>]*>.-</script>", "")

    -- Remove style tags
    text = string.gsub(text, "<style[^>]*>.-</style>", "")

    -- Remove onclick and other event handlers
    text = string.gsub(text, "on%w+=[\"'][^\"']*[\"']", "")

    return text
end

-- Normalize markdown links
function normalize_markdown_links(text)
    -- Convert [text](url) to consistent format
    text = string.gsub(text, "%[([^%]]+)%]%s*%(([^%)]+)%)", "[%1](%2)")

    return text
end

-- Truncate text to maximum length
function truncate_text(text, max_length)
    if #text > max_length then
        return string.sub(text, 1, max_length) .. "..."
    end
    return text
end

-- Extract domain from URL
function extract_domain(url)
    return string.match(url, "https?://([^/]+)")
end

-- Format timestamp as ISO 8601
function format_timestamp(timestamp)
    return os.date("%Y-%m-%dT%H:%M:%SZ", timestamp)
end
