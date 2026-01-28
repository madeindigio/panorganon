-- security-filter.lua
-- Security-focused filters for protecting sensitive data

-- ============================================
-- Global Security Input Filter
-- ============================================

_G["global-input"] = function(context)
    local params = context.parameters

    -- Block searches for sensitive patterns
    if params.query then
        local sensitive_patterns = {
            "password",
            "api[_-]?key",
            "secret",
            "token",
            "credential",
            "private[_-]?key"
        }

        for _, pattern in ipairs(sensitive_patterns) do
            if string.match(string.lower(params.query), pattern) then
                error(string.format("Search for sensitive information blocked: %s", pattern))
            end
        end
    end

    -- Validate and sanitize URLs to prevent SSRF
    if params.url then
        local blocked_patterns = {
            "localhost",
            "127%.0%.0%.1",
            "192%.168%.",
            "10%.%d+%.",
            "172%.1[6-9]%.",
            "172%.2[0-9]%.",
            "172%.3[0-1]%.",
            "169%.254%.",     -- Link-local
            "::1",            -- IPv6 localhost
            "file://",        -- File protocol
            "gopher://",      -- Gopher protocol
        }

        for _, pattern in ipairs(blocked_patterns) do
            if string.match(params.url, pattern) then
                error(string.format("Blocked URL pattern: %s", pattern))
            end
        end
    end

    -- Limit parameter sizes to prevent DoS
    if params.content and #params.content > 100000 then
        error("Content size exceeds maximum allowed (100KB)")
    end

    if params.query and #params.query > 10000 then
        error("Query size exceeds maximum allowed (10KB)")
    end

    return params
end

-- ============================================
-- Global Security Output Filter
-- ============================================

_G["global-output"] = function(context)
    local result = context.result

    if not result.content then
        return result
    end

    -- Redact sensitive data patterns in all text content
    for i, item in ipairs(result.content) do
        if item.type == "text" and item.text then
            -- Redact API keys
            item.text = string.gsub(item.text, "sk%-[a-zA-Z0-9_-]+", "sk-[REDACTED]")
            item.text = string.gsub(item.text, "sk%-ant%-[a-zA-Z0-9%-]+", "sk-ant-[REDACTED]")

            -- Redact Bearer tokens
            item.text = string.gsub(item.text, "[Bb]earer [a-zA-Z0-9_%.%-]+", "Bearer [REDACTED]")

            -- Redact AWS keys
            item.text = string.gsub(item.text, "AKIA[0-9A-Z]{16}", "AKIA[REDACTED]")

            -- Redact JWT tokens
            item.text = string.gsub(item.text, "eyJ[a-zA-Z0-9_-]*%.eyJ[a-zA-Z0-9_-]*%.[a-zA-Z0-9_-]*", "jwt.[REDACTED]")

            -- Redact email addresses (optional, uncomment if needed)
            -- item.text = string.gsub(item.text, "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+%.[a-zA-Z]{2,}", "[EMAIL-REDACTED]")

            -- Redact IP addresses (optional, uncomment if needed)
            -- item.text = string.gsub(item.text, "%d+%.%d+%.%d+%.%d+", "[IP-REDACTED]")

            -- Redact system paths
            item.text = string.gsub(item.text, "/home/[^/]+", "/home/[USER]")
            item.text = string.gsub(item.text, "/Users/[^/]+", "/Users/[USER]")
            item.text = string.gsub(item.text, "C:\\Users\\[^\\]+", "C:\\Users\\[USER]")
            item.text = string.gsub(item.text, "C:\\Documents and Settings\\[^\\]+", "C:\\Documents and Settings\\[USER]")
        end
    end

    return result
end

-- ============================================
-- Helper Functions
-- ============================================

-- Check if a string contains any of the patterns
function contains_pattern(text, patterns)
    for _, pattern in ipairs(patterns) do
        if string.match(text, pattern) then
            return true
        end
    end
    return false
end

-- Validate email format
function is_valid_email(email)
    return string.match(email, "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+%.[a-zA-Z]{2,}$") ~= nil
end

-- Validate URL format
function is_valid_url(url)
    return string.match(url, "^https?://[%w-_%.]+") ~= nil
end
