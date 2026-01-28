-- test-filters.lua
-- Test Lua filters for unit testing

-- Note: Lua function names cannot contain hyphens, so we use the approach
-- of defining them with underscores and the FilterManager will normalize
-- server names to use hyphens when looking up functions

-- This is accessed as "test-server-input" by the FilterManager
_G["test-server-input"] = function(context)
    local params = context.parameters

    -- Add test modification
    params.filtered = true
    params.timestamp = context.timestamp

    return params
end

_G["test-server-output"] = function(context)
    local result = context.result

    -- Add test metadata
    result.filtered_output = true
    result.duration = context.duration

    return result
end

_G["error-server-input"] = function(context)
    error("Intentional error for testing")
end

_G["timeout-server-input"] = function(context)
    -- Sleep for a long time to trigger timeout
    local start = os.time()
    while os.time() - start < 10 do
        -- busy wait
    end
    return context.parameters
end

_G["global-input"] = function(context)
    local params = context.parameters
    params.global_filter = true
    return params
end

_G["global-output"] = function(context)
    local result = context.result
    result.global_filter = true
    return result
end
