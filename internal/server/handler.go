package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sevir/panorganon/internal/database"
	"github.com/sevir/panorganon/internal/downstream"
	"github.com/sevir/panorganon/internal/tools"
	"go.uber.org/zap"
)

// ToolHandlers contains references to services that implement tool logic
type ToolHandlers struct {
	Logger           *zap.Logger
	Manager          *downstream.Manager
	DB               *database.DB
	DiscoveryService *tools.DiscoveryService
	SearchService    *tools.SearchService
	ExecutorService  *tools.ExecutorService
}

// NewMCPServer creates and configures a new MCP server with tool handlers
func NewMCPServer(handlers *ToolHandlers) *server.MCPServer {
	s := server.NewMCPServer(
		"Panorganon",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register search_tools tool
	searchToolsTool := mcp.NewTool("search_tools",
		mcp.WithDescription("Search for appropriate tools based on a task description. Returns relevant tools from downstream MCP servers that can help accomplish the specified task."),
		mcp.WithString("task_description",
			mcp.Required(),
			mcp.Description("Natural language description of the task to accomplish"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of tools to return (default: 10)"),
		),
	)

	s.AddTool(searchToolsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlers.Logger.Info("search_tools called", zap.Any("arguments", request.Params.Arguments))

		if handlers.SearchService == nil {
			return mcp.NewToolResultError("Search service not initialized"), nil
		}

		// Extract arguments
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Invalid arguments format"), nil
		}

		taskDescription, ok := args["task_description"].(string)
		if !ok || taskDescription == "" {
			return mcp.NewToolResultError("task_description is required"), nil
		}

		maxResults := 10 // default
		if mr, ok := args["max_results"].(float64); ok {
			maxResults = int(mr)
		}

		// Perform search
		results, err := handlers.SearchService.SearchTools(ctx, taskDescription, maxResults)
		if err != nil {
			errMsg := fmt.Sprintf("Search failed: %v", err)
			handlers.Logger.Error(errMsg)
			return mcp.NewToolResultError(errMsg), nil
		}

		// Convert to JSON
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal results: %v", err)), nil
		}

		handlers.Logger.Info("Search completed", zap.Int("results", len(results)))
		return mcp.NewToolResultText(string(data)), nil
	})

	// Register exec_tool tool
	execToolTool := mcp.NewTool("exec_tool",
		mcp.WithDescription("Execute a tool from a downstream MCP server. This is the primary way to invoke tools discovered through search_tools."),
		mcp.WithString("tool_name",
			mcp.Required(),
			mcp.Description("Name of the tool to execute"),
		),
		mcp.WithObject("parameters",
			mcp.Required(),
			mcp.Description("JSON object with tool parameters matching the tool's input schema"),
		),
		mcp.WithString("server_name",
			mcp.Description("Explicit server name if the tool exists in multiple servers (optional)"),
		),
	)

	s.AddTool(execToolTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlers.Logger.Info("exec_tool called", zap.Any("arguments", request.Params.Arguments))

		if handlers.ExecutorService == nil {
			return mcp.NewToolResultError("Executor service not initialized"), nil
		}

		// Extract arguments
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Invalid arguments format"), nil
		}

		toolName, ok := args["tool_name"].(string)
		if !ok || toolName == "" {
			return mcp.NewToolResultError("tool_name is required"), nil
		}

		parameters, ok := args["parameters"].(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("parameters must be an object"), nil
		}

		serverName := ""
		if sn, ok := args["server_name"].(string); ok {
			serverName = sn
		}

		// Execute the tool
		result, err := handlers.ExecutorService.ExecTool(ctx, toolName, parameters, serverName)
		if err != nil {
			errMsg := fmt.Sprintf("Tool execution failed: %v", err)
			handlers.Logger.Error(errMsg,
				zap.String("tool", toolName),
				zap.String("server", serverName),
			)
			return mcp.NewToolResultError(errMsg), nil
		}

		if !result.Success {
			return mcp.NewToolResultError(result.Error), nil
		}

		// Return the actual tool result
		return result.Result, nil
	})

	// Register list_servers tool
	listServersTool := mcp.NewTool("list_servers",
		mcp.WithDescription("List all configured downstream MCP servers with their status information (running, stopped, available tools count)."),
	)

	s.AddTool(listServersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlers.Logger.Info("list_servers called")

		servers := handlers.Manager.ListServers()

		// Convert to JSON
		data, err := json.MarshalIndent(servers, "", "  ")
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	// Register refresh_tools tool
	refreshToolsTool := mcp.NewTool("refresh_tools",
		mcp.WithDescription("Force refresh of the tool metadata cache by re-discovering all tools from downstream servers. Useful after adding or updating downstream servers."),
		mcp.WithString("server_name",
			mcp.Description("Optional: refresh only tools from a specific server"),
		),
	)

	s.AddTool(refreshToolsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlers.Logger.Info("refresh_tools called", zap.Any("arguments", request.Params.Arguments))

		if handlers.DiscoveryService == nil {
			return mcp.NewToolResultError("Discovery service not initialized"), nil
		}

		// Extract server_name from arguments if provided
		var serverName string
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if name, ok := args["server_name"].(string); ok {
				serverName = name
			}
		}

		// Perform discovery
		var result string
		var err error

		if serverName != "" {
			// Refresh specific server
			handlers.Logger.Info("Refreshing tools for specific server", zap.String("server", serverName))
			count, err := handlers.DiscoveryService.DiscoverServer(ctx, serverName)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to refresh tools from server %s: %v", serverName, err)
				handlers.Logger.Error(errMsg)
				return mcp.NewToolResultError(errMsg), nil
			}
			result = fmt.Sprintf("Successfully refreshed %d tools from server '%s'", count, serverName)
		} else {
			// Refresh all servers
			handlers.Logger.Info("Refreshing tools from all servers")
			err = handlers.DiscoveryService.DiscoverAll(ctx)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to refresh tools: %v", err)
				handlers.Logger.Error(errMsg)
				return mcp.NewToolResultError(errMsg), nil
			}

			// Get metrics to report results
			metrics := handlers.DiscoveryService.GetMetrics()
			result = fmt.Sprintf(
				"Successfully refreshed tools from all servers\nTotal tools: %d\nFailed servers: %d\nDuration: %s",
				metrics.TotalToolsDiscovered,
				metrics.FailedDiscoveries,
				metrics.DiscoveryDuration,
			)
		}

		handlers.Logger.Info("Tool refresh completed", zap.String("result", result))
		return mcp.NewToolResultText(result), nil
	})

	return s
}
