package luafilters

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// FilterManager manages Lua filter execution
type FilterManager struct {
	// scriptPath is the path to the Lua filter script
	scriptPath string

	// L is the Lua state
	L *lua.LState

	// enabled indicates whether filters are enabled
	enabled bool

	// timeout is the maximum execution time for filters
	timeout time.Duration

	// strictMode determines behavior on filter errors
	strictMode bool

	// logFilteredData determines whether to log filtered data
	logFilteredData bool

	// logger is the structured logger
	logger *zap.Logger

	// mu protects concurrent access to the Lua state
	mu sync.RWMutex

	// scriptLoaded indicates whether the script has been loaded
	scriptLoaded bool
}

// FilterManagerConfig holds configuration for the FilterManager
type FilterManagerConfig struct {
	ScriptPath      string
	Enabled         bool
	Timeout         time.Duration
	StrictMode      bool
	LogFilteredData bool
	Logger          *zap.Logger
}

// NewFilterManager creates a new FilterManager instance
func NewFilterManager(config FilterManagerConfig) (*FilterManager, error) {
	if config.Logger == nil {
		// Create a default logger if none provided
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
		config.Logger = logger
	}

	fm := &FilterManager{
		scriptPath:      config.ScriptPath,
		enabled:         config.Enabled,
		timeout:         config.Timeout,
		strictMode:      config.StrictMode,
		logFilteredData: config.LogFilteredData,
		logger:          config.Logger,
	}

	// Initialize Lua state if filters are enabled
	if fm.enabled {
		if err := fm.initLuaState(); err != nil {
			return nil, fmt.Errorf("failed to initialize Lua state: %w", err)
		}

		// Load the script if path is provided
		if fm.scriptPath != "" {
			if err := fm.LoadScript(); err != nil {
				if fm.strictMode {
					return nil, fmt.Errorf("failed to load filter script: %w", err)
				}
				// In non-strict mode, log the error but continue
				fm.logger.Warn("Failed to load filter script, continuing without filters",
					zap.String("script_path", fm.scriptPath),
					zap.Error(err))
			}
		}
	}

	return fm, nil
}

// initLuaState initializes the Lua state
func (fm *FilterManager) initLuaState() error {
	L, err := InitLuaState()
	if err != nil {
		return err
	}
	fm.L = L
	return nil
}

// LoadScript loads the Lua filter script from the configured path
func (fm *FilterManager) LoadScript() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.scriptPath == "" {
		return fmt.Errorf("no script path configured")
	}

	// Check if file exists
	if _, err := os.Stat(fm.scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script file does not exist: %s", fm.scriptPath)
	}

	// Load the script
	if err := fm.L.DoFile(fm.scriptPath); err != nil {
		return fmt.Errorf("failed to load script %s: %w", fm.scriptPath, err)
	}

	fm.scriptLoaded = true
	fm.logger.Info("Loaded Lua filter script",
		zap.String("script_path", fm.scriptPath))

	return nil
}

// ReloadScript reloads the Lua filter script
func (fm *FilterManager) ReloadScript() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.scriptPath == "" {
		return fmt.Errorf("no script path configured")
	}

	// Close the old Lua state
	if fm.L != nil {
		CloseLuaState(fm.L)
		fm.L = nil
	}

	// Create a new Lua state
	L, err := InitLuaState()
	if err != nil {
		return fmt.Errorf("failed to reinitialize Lua state: %w", err)
	}
	fm.L = L

	// Reload the script
	if err := fm.L.DoFile(fm.scriptPath); err != nil {
		return fmt.Errorf("failed to reload script %s: %w", fm.scriptPath, err)
	}

	fm.scriptLoaded = true
	fm.logger.Info("Reloaded Lua filter script",
		zap.String("script_path", fm.scriptPath))

	return nil
}

// ApplyInputFilter applies input filters to parameters before execution
func (fm *FilterManager) ApplyInputFilter(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
	if !fm.enabled || !fm.scriptLoaded {
		// Return original parameters unchanged
		result := NewHookResult()
		result.Data = hookCtx.Parameters
		return result, nil
	}

	// Build function name: <server-name>-input
	functionName := fm.buildFunctionName(hookCtx.ServerName, FilterTypeInput)

	return fm.executeFilter(ctx, functionName, hookCtx)
}

// ApplyOutputFilter applies output filters to results after execution
func (fm *FilterManager) ApplyOutputFilter(ctx context.Context, hookCtx *HookContext) (*HookResult, error) {
	if !fm.enabled || !fm.scriptLoaded {
		// Return original result unchanged
		result := NewHookResult()
		result.Data = hookCtx.Result
		return result, nil
	}

	// Build function name: <server-name>-output
	functionName := fm.buildFunctionName(hookCtx.ServerName, FilterTypeOutput)

	return fm.executeFilter(ctx, functionName, hookCtx)
}

// buildFunctionName builds the Lua function name based on server name and filter type
// Format: <server-name>-input or <server-name>-output
func (fm *FilterManager) buildFunctionName(serverName string, filterType FilterType) string {
	// Replace underscores and dots with hyphens for consistency
	normalized := strings.ReplaceAll(serverName, "_", "-")
	normalized = strings.ReplaceAll(normalized, ".", "-")
	return fmt.Sprintf("%s-%s", normalized, filterType)
}

// executeFilter executes a Lua filter function with timeout
func (fm *FilterManager) executeFilter(ctx context.Context, functionName string, hookCtx *HookContext) (*HookResult, error) {
	startTime := time.Now()
	result := NewHookResult()

	fm.mu.RLock()
	defer fm.mu.RUnlock()

	// Check if the function exists
	fn := fm.L.GetGlobal(functionName)
	if fn == lua.LNil {
		// Function doesn't exist - try global fallback
		fallbackName := fmt.Sprintf("global-%s", hookCtx.FilterType)
		fn = fm.L.GetGlobal(fallbackName)
		if fn == lua.LNil {
			// No filter found - return original data
			if hookCtx.FilterType == FilterTypeInput {
				result.Data = hookCtx.Parameters
			} else {
				result.Data = hookCtx.Result
			}
			return result, nil
		}
		functionName = fallbackName
	}

	// Create context table to pass to Lua
	contextTable := fm.createContextTable(hookCtx)

	// Execute with timeout
	done := make(chan bool, 1)
	var execErr error

	go func() {
		// Call the Lua function
		if err := fm.L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    1,
			Protect: true,
		}, contextTable); err != nil {
			execErr = err
			done <- true
			return
		}

		// Get the return value
		ret := fm.L.Get(-1)
		fm.L.Pop(1)

		if ret == lua.LNil {
			execErr = fmt.Errorf("filter function %s returned nil", functionName)
			done <- true
			return
		}

		// Convert the returned table to Go map
		if retTable, ok := ret.(*lua.LTable); ok {
			result.Data = luaTableToMap(retTable)
			result.Modified = true
		} else {
			execErr = fmt.Errorf("filter function %s did not return a table", functionName)
		}

		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		if execErr != nil {
			result.SetError(execErr, fmt.Sprintf("Filter execution error: %v", execErr))
			fm.logger.Error("Filter execution failed",
				zap.String("function", functionName),
				zap.Error(execErr))

			if fm.strictMode {
				return result, execErr
			}

			// In non-strict mode, return original data
			if hookCtx.FilterType == FilterTypeInput {
				result.Data = hookCtx.Parameters
			} else {
				result.Data = hookCtx.Result
			}
			result.Modified = false
		}

	case <-time.After(fm.timeout):
		err := fmt.Errorf("filter function %s timed out after %v", functionName, fm.timeout)
		result.SetError(err, err.Error())
		fm.logger.Error("Filter execution timeout",
			zap.String("function", functionName),
			zap.Duration("timeout", fm.timeout))

		if fm.strictMode {
			return result, err
		}

		// In non-strict mode, return original data
		if hookCtx.FilterType == FilterTypeInput {
			result.Data = hookCtx.Parameters
		} else {
			result.Data = hookCtx.Result
		}
		result.Modified = false

	case <-ctx.Done():
		err := ctx.Err()
		result.SetError(err, "Context cancelled")
		return result, err
	}

	result.ExecutionTime = time.Since(startTime)

	// Log filtered data if enabled
	if fm.logFilteredData && result.Modified {
		fm.logger.Debug("Filter applied",
			zap.String("function", functionName),
			zap.Duration("execution_time", result.ExecutionTime),
			zap.Any("data", result.Data))
	}

	return result, nil
}

// createContextTable creates a Lua table from HookContext
func (fm *FilterManager) createContextTable(hookCtx *HookContext) *lua.LTable {
	table := fm.L.NewTable()

	SetTableField(fm.L, table, "server_name", hookCtx.ServerName)
	SetTableField(fm.L, table, "tool_name", hookCtx.ToolName)
	SetTableField(fm.L, table, "request_id", hookCtx.RequestID)
	SetTableField(fm.L, table, "timestamp", hookCtx.Timestamp)

	if hookCtx.FilterType == FilterTypeInput && hookCtx.Parameters != nil {
		SetTableField(fm.L, table, "parameters", hookCtx.Parameters)
	}

	if hookCtx.FilterType == FilterTypeOutput && hookCtx.Result != nil {
		SetTableField(fm.L, table, "result", hookCtx.Result)
		SetTableField(fm.L, table, "duration", hookCtx.Duration)
	}

	return table
}

// Close closes the FilterManager and releases resources
func (fm *FilterManager) Close() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.L != nil {
		CloseLuaState(fm.L)
		fm.L = nil
	}
}

// IsEnabled returns whether filters are enabled
func (fm *FilterManager) IsEnabled() bool {
	return fm.enabled
}

// IsScriptLoaded returns whether a script has been loaded
func (fm *FilterManager) IsScriptLoaded() bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.scriptLoaded
}
