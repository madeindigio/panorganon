package downstream

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// DownstreamClient represents a client for communicating with a downstream MCP server
type DownstreamClient interface {
	// Start starts the downstream server
	Start(ctx context.Context) error

	// Stop stops the downstream server
	Stop() error

	// IsRunning returns true if the server is currently running
	IsRunning() bool

	// CallTool calls a tool on the downstream server
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error)

	// ListTools returns all tools provided by this server
	ListTools(ctx context.Context) ([]mcp.Tool, error)

	// GetName returns the server name
	GetName() string

	// GetType returns the server type (stdio, streamable-http, sse)
	GetType() string
}
