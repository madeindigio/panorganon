package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sevir/panorganon/internal/database"
	"github.com/sevir/panorganon/internal/downstream"
	"github.com/sevir/panorganon/internal/luafilters"
	"go.uber.org/zap"
)

// ExecutionResult holds information about a tool execution
type ExecutionResult struct {
	ToolName  string                 `json:"tool_name"`
	Server    string                 `json:"server"`
	Success   bool                   `json:"success"`
	Duration  time.Duration          `json:"duration"`
	Result    *mcp.CallToolResult    `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Retries   int                    `json:"retries"`
}

// ExecutorService handles tool execution on downstream servers
type ExecutorService struct {
	manager        *downstream.Manager
	db             *database.DB
	logger         *zap.Logger
	filterManager  *luafilters.FilterManager
	defaultTimeout time.Duration
	maxRetries     int
}

// NewExecutorService creates a new executor service
func NewExecutorService(manager *downstream.Manager, db *database.DB, logger *zap.Logger, filterManager *luafilters.FilterManager) *ExecutorService {
	return &ExecutorService{
		manager:        manager,
		db:             db,
		logger:         logger.With(zap.String("component", "executor")),
		filterManager:  filterManager,
		defaultTimeout: 60 * time.Second,
		maxRetries:     3,
	}
}

// ExecTool executes a tool on a downstream server
func (e *ExecutorService) ExecTool(ctx context.Context, toolName string, parameters map[string]interface{}, serverName string) (*ExecutionResult, error) {
	startTime := time.Now()

	e.logger.Info("Executing tool",
		zap.String("tool", toolName),
		zap.String("server", serverName),
	)

	// Lookup tool in database
	tool, server, err := e.lookupTool(toolName, serverName)
	if err != nil {
		return &ExecutionResult{
			ToolName: toolName,
			Server:   serverName,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    err.Error(),
		}, err
	}

	e.logger.Debug("Tool found in database",
		zap.String("tool", toolName),
		zap.String("server", server),
	)

	// Validate parameters against input schema
	if err := e.validateParameters(parameters, tool.InputSchema); err != nil {
		return &ExecutionResult{
			ToolName: toolName,
			Server:   server,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    fmt.Sprintf("parameter validation failed: %v", err),
		}, fmt.Errorf("parameter validation failed: %w", err)
	}

	// 🔵 HOOK POINT 1: Apply input filter before execution
	if e.filterManager != nil && e.filterManager.IsEnabled() {
		requestID := uuid.New().String()
		hookCtx := luafilters.NewInputContext(server, toolName, parameters, requestID)

		filterResult, filterErr := e.filterManager.ApplyInputFilter(ctx, hookCtx)
		if filterErr != nil {
			e.logger.Error("Input filter error",
				zap.String("tool", toolName),
				zap.String("server", server),
				zap.Error(filterErr),
			)
			return &ExecutionResult{
				ToolName: toolName,
				Server:   server,
				Success:  false,
				Duration: time.Since(startTime),
				Error:    fmt.Sprintf("input filter error: %v", filterErr),
			}, fmt.Errorf("input filter error: %w", filterErr)
		}

		if filterResult.Modified {
			e.logger.Debug("Input parameters filtered",
				zap.String("tool", toolName),
				zap.String("server", server),
				zap.Duration("filter_time", filterResult.ExecutionTime),
			)
			// Use filtered parameters for execution
			parameters = filterResult.Data
		}
	}

	// Execute with retries
	var result *mcp.CallToolResult
	var execErr error
	retries := 0

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		if attempt > 0 {
			retries++
			e.logger.Info("Retrying tool execution",
				zap.String("tool", toolName),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", e.maxRetries),
			)
			// Wait before retry
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		result, execErr = e.executeTool(ctx, server, toolName, parameters)
		if execErr == nil {
			break
		}

		// Check if error is retryable
		if !e.isRetryableError(execErr) {
			e.logger.Warn("Non-retryable error, stopping retries", zap.Error(execErr))
			break
		}
	}

	duration := time.Since(startTime)

	if execErr != nil {
		e.logger.Error("Tool execution failed",
			zap.String("tool", toolName),
			zap.String("server", server),
			zap.Error(execErr),
			zap.Int("retries", retries),
			zap.Duration("duration", duration),
		)

		return &ExecutionResult{
			ToolName: toolName,
			Server:   server,
			Success:  false,
			Duration: duration,
			Error:    execErr.Error(),
			Retries:  retries,
		}, execErr
	}

	e.logger.Info("Tool execution successful",
		zap.String("tool", toolName),
		zap.String("server", server),
		zap.Int("retries", retries),
		zap.Duration("duration", duration),
	)

	return &ExecutionResult{
		ToolName: toolName,
		Server:   server,
		Success:  true,
		Duration: duration,
		Result:   result,
		Retries:  retries,
	}, nil
}

// lookupTool finds the tool in the database and determines the server
func (e *ExecutorService) lookupTool(toolName string, serverName string) (*database.ToolRecord, string, error) {
	if serverName != "" {
		// Look for tool on specific server
		tool, err := e.db.GetTool(toolName)
		if err != nil {
			return nil, "", fmt.Errorf("failed to query tool: %w", err)
		}
		if tool == nil || tool.ServerID != serverName {
			return nil, "", fmt.Errorf("tool %s not found on server %s", toolName, serverName)
		}
		return tool, serverName, nil
	}

	// Look for tool on any server
	tool, err := e.db.GetTool(toolName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query tool: %w", err)
	}
	if tool == nil {
		return nil, "", fmt.Errorf("tool %s not found", toolName)
	}

	return tool, tool.ServerID, nil
}

// validateParameters validates parameters against the tool's input schema
func (e *ExecutorService) validateParameters(parameters map[string]interface{}, schemaJSON string) error {
	// Parse schema
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		e.logger.Warn("Failed to parse input schema, skipping validation", zap.Error(err))
		return nil // Don't fail on schema parse errors
	}

	// Basic validation: check required properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil // No properties defined, skip validation
	}

	required, ok := schema["required"].([]interface{})
	if ok {
		for _, req := range required {
			reqStr, ok := req.(string)
			if !ok {
				continue
			}
			if _, exists := parameters[reqStr]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", reqStr)
			}
		}
	}

	// Basic type validation for provided parameters
	for paramName, paramValue := range parameters {
		propSchema, exists := properties[paramName].(map[string]interface{})
		if !exists {
			continue // Unknown parameter, let the downstream server handle it
		}

		paramType, ok := propSchema["type"].(string)
		if !ok {
			continue // No type specified
		}

		// Check type match
		if !e.validateType(paramValue, paramType) {
			return fmt.Errorf("parameter '%s' has wrong type: expected %s", paramName, paramType)
		}
	}

	return nil
}

// validateType checks if a value matches the expected JSON schema type
func (e *ExecutorService) validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number", "integer":
		_, ok := value.(float64)
		if !ok {
			_, ok = value.(int)
		}
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	default:
		return true // Unknown type, allow it
	}
}

// executeTool performs the actual tool execution
func (e *ExecutorService) executeTool(ctx context.Context, serverName string, toolName string, parameters map[string]interface{}) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Get or start the server
	client, err := e.manager.GetOrStart(ctx, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.defaultTimeout)
	defer cancel()

	// Call the tool
	result, err := client.CallTool(execCtx, toolName, parameters)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	duration := time.Since(startTime)

	// 🔵 HOOK POINT 2: Apply output filter after execution
	if e.filterManager != nil && e.filterManager.IsEnabled() {
		requestID := uuid.New().String()

		// Convert result to map for filtering
		resultMap := resultToMap(result)

		// Create hook context
		hookCtx := luafilters.NewOutputContext(serverName, toolName, resultMap, requestID, duration)

		// Apply output filter
		filterResult, filterErr := e.filterManager.ApplyOutputFilter(ctx, hookCtx)
		if filterErr != nil {
			e.logger.Error("Output filter error",
				zap.String("tool", toolName),
				zap.String("server", serverName),
				zap.Error(filterErr),
			)
			// In case of filter error, return the original result
			// unless strict mode would have prevented this earlier
		} else if filterResult.Modified {
			e.logger.Debug("Output result filtered",
				zap.String("tool", toolName),
				zap.String("server", serverName),
				zap.Duration("filter_time", filterResult.ExecutionTime),
			)
			// Convert filtered map back to mcp.CallToolResult
			result = mapToResult(filterResult.Data)
		}
	}

	// Stop server if not keepalive
	serverInfo := e.manager.GetServerInfo(serverName)
	if serverInfo != nil && !serverInfo.KeepAlive {
		e.logger.Debug("Stopping non-keepalive server after execution", zap.String("server", serverName))
		if stopErr := e.manager.Stop(serverName); stopErr != nil {
			e.logger.Warn("Failed to stop server after execution",
				zap.String("server", serverName),
				zap.Error(stopErr),
			)
		}
	}

	return result, nil
}

// isRetryableError determines if an error is retryable
func (e *ExecutorService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retryable errors: timeouts, connection issues, temporary failures
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"server not running",
		"failed to start server",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// resultToMap converts mcp.CallToolResult to a map for Lua filters
func resultToMap(result *mcp.CallToolResult) map[string]interface{} {
	if result == nil {
		return make(map[string]interface{})
	}

	m := make(map[string]interface{})

	// Convert content array
	if result.Content != nil {
		contentArray := make([]interface{}, len(result.Content))
		for i, item := range result.Content {
			contentItem := make(map[string]interface{})

			switch v := item.(type) {
			case mcp.TextContent:
				contentItem["type"] = "text"
				contentItem["text"] = v.Text
			case mcp.ImageContent:
				contentItem["type"] = "image"
				contentItem["data"] = v.Data
				contentItem["mimeType"] = v.MIMEType
			case mcp.EmbeddedResource:
				contentItem["type"] = "resource"
				// Add resource fields as needed
			default:
				contentItem["type"] = "unknown"
			}

			contentArray[i] = contentItem
		}
		m["content"] = contentArray
	}

	// Add is_error field
	m["isError"] = result.IsError

	return m
}

// mapToResult converts a filtered map back to mcp.CallToolResult
func mapToResult(m map[string]interface{}) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}

	// Convert content array
	if contentVal, ok := m["content"].([]interface{}); ok {
		content := make([]mcp.Content, 0, len(contentVal))
		for _, item := range contentVal {
			if itemMap, ok := item.(map[string]interface{}); ok {
				itemType, _ := itemMap["type"].(string)

				switch itemType {
				case "text":
					if text, ok := itemMap["text"].(string); ok {
						content = append(content, mcp.TextContent{
							Type: "text",
							Text: text,
						})
					}
				case "image":
					imageContent := mcp.ImageContent{
						Type: "image",
					}
					if data, ok := itemMap["data"].(string); ok {
						imageContent.Data = data
					}
					if mimeType, ok := itemMap["mimeType"].(string); ok {
						imageContent.MIMEType = mimeType
					}
					content = append(content, imageContent)
				default:
					// For unknown types, try to preserve as text
					if text, ok := itemMap["text"].(string); ok {
						content = append(content, mcp.TextContent{
							Type: "text",
							Text: text,
						})
					}
				}
			}
		}
		result.Content = content
	}

	// Convert isError field
	if isError, ok := m["isError"].(bool); ok {
		result.IsError = isError
	}

	return result
}
