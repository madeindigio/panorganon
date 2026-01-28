package downstream

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
)

// StdioClient implements DownstreamClient for stdio-based MCP servers
type StdioClient struct {
	config    config.DownstreamServer
	logger    *zap.Logger
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	running   bool
	mu        sync.RWMutex
	rpcClient *JSONRPCClient
}

// NewStdioClient creates a new stdio client
func NewStdioClient(cfg config.DownstreamServer, logger *zap.Logger) *StdioClient {
	return &StdioClient{
		config:  cfg,
		logger:  logger.With(zap.String("server", cfg.Name), zap.String("type", "stdio")),
		running: false,
	}
}

// Start starts the downstream server process
func (c *StdioClient) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		c.logger.Debug("Server already running")
		return nil
	}

	c.logger.Info("Starting stdio server", zap.String("command", c.config.Command))

	// Create command with args
	c.cmd = exec.CommandContext(ctx, c.config.Command, c.config.Args...)

	// Set environment variables
	if c.config.Env != nil {
		for k, v := range c.config.Env {
			c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Get stdin and stdout pipes
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Initialize JSON-RPC client
	c.rpcClient = NewJSONRPCClient(c.stdout, c.stdin, c.logger)
	c.rpcClient.Start()

	c.running = true
	c.logger.Info("Server started successfully")

	// Monitor process in background
	go c.monitorProcess()

	return nil
}

// Stop stops the downstream server process
func (c *StdioClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.logger.Info("Stopping stdio server")

	// Stop JSON-RPC client
	if c.rpcClient != nil {
		c.rpcClient.Stop()
	}

	// Close stdin to signal shutdown
	if c.stdin != nil {
		c.stdin.Close()
	}

	// Wait for process to exit
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Wait(); err != nil {
			c.logger.Warn("Process exited with error", zap.Error(err))
		}
	}

	c.running = false
	c.logger.Info("Server stopped")
	return nil
}

// IsRunning returns true if the server is running
func (c *StdioClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// CallTool calls a tool on the downstream server
func (c *StdioClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("server is not running")
	}

	if c.rpcClient == nil {
		return nil, fmt.Errorf("JSON-RPC client not initialized")
	}

	c.logger.Debug("Calling tool", zap.String("tool", name))

	result, err := c.rpcClient.CallToolRPC(ctx, name, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	return result, nil
}

// ListTools returns all tools from the downstream server
func (c *StdioClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("server is not running")
	}

	if c.rpcClient == nil {
		return nil, fmt.Errorf("JSON-RPC client not initialized")
	}

	c.logger.Debug("Listing tools")

	tools, err := c.rpcClient.ListToolsRPC(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return tools, nil
}

// GetName returns the server name
func (c *StdioClient) GetName() string {
	return c.config.Name
}

// GetType returns the server type
func (c *StdioClient) GetType() string {
	return c.config.Type
}

// monitorProcess monitors the process and restarts if keepalive is enabled
func (c *StdioClient) monitorProcess() {
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}

	// Wait for process to exit
	err := c.cmd.Wait()

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()

	if err != nil {
		c.logger.Error("Process exited with error", zap.Error(err))
	} else {
		c.logger.Info("Process exited normally")
	}

	// TODO: Implement restart logic for keepalive servers
	if c.config.KeepAlive {
		c.logger.Info("KeepAlive enabled - restart logic to be implemented")
	}
}
