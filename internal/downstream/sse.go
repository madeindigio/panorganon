package downstream

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
)

// SSEClient implements DownstreamClient for SSE-based MCP servers
type SSEClient struct {
	config  config.DownstreamServer
	logger  *zap.Logger
	running bool
	mu      sync.RWMutex
}

// NewSSEClient creates a new SSE client
func NewSSEClient(cfg config.DownstreamServer, logger *zap.Logger) *SSEClient {
	return &SSEClient{
		config:  cfg,
		logger:  logger.With(zap.String("server", cfg.Name), zap.String("type", "sse")),
		running: false,
	}
}

// Start establishes connection to the SSE server
func (c *SSEClient) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Starting SSE client", zap.String("url", c.config.URL))
	// TODO: Implement SSE connection
	c.running = true
	return nil
}

// Stop closes the SSE connection
func (c *SSEClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Stopping SSE client")
	c.running = false
	return nil
}

// IsRunning returns true if connected
func (c *SSEClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// CallTool calls a tool via SSE
func (c *SSEClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("client is not connected")
	}

	c.logger.Debug("Calling tool", zap.String("tool", name))
	// TODO: Implement SSE tool call
	return nil, fmt.Errorf("SSE tool call not yet implemented")
}

// ListTools returns all tools from the SSE server
func (c *SSEClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("client is not connected")
	}

	c.logger.Debug("Listing tools")
	// TODO: Implement SSE list tools
	return nil, fmt.Errorf("SSE list tools not yet implemented")
}

// GetName returns the server name
func (c *SSEClient) GetName() string {
	return c.config.Name
}

// GetType returns the server type
func (c *SSEClient) GetType() string {
	return c.config.Type
}
