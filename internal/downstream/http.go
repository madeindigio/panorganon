package downstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
)

// HTTPClient implements DownstreamClient for HTTP-based MCP servers
type HTTPClient struct {
	config     config.DownstreamServer
	logger     *zap.Logger
	httpClient *http.Client
	running    bool
	mu         sync.RWMutex
	nextID     atomic.Int64
	cmd        *exec.Cmd
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient(cfg config.DownstreamServer, logger *zap.Logger) *HTTPClient {
	return &HTTPClient{
		config:     cfg,
		logger:     logger.With(zap.String("server", cfg.Name), zap.String("type", "streamable-http")),
		httpClient: &http.Client{},
		running:    false,
	}
}

// Start launches the server process (if start_cmd is configured) and marks the HTTP client as ready
func (c *HTTPClient) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Launch the server process if start_cmd is defined
	if c.config.StartCmd != "" {
		c.logger.Info("Launching server process",
			zap.String("start_cmd", c.config.StartCmd),
			zap.Strings("start_args", c.config.StartArgs),
		)

		c.cmd = exec.CommandContext(ctx, c.config.StartCmd, c.config.StartArgs...)

		// Set environment variables
		if c.config.Env != nil {
			for k, v := range c.config.Env {
				c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		if err := c.cmd.Start(); err != nil {
			return fmt.Errorf("failed to start server process: %w", err)
		}

		// Monitor the process in the background
		go c.monitorProcess()

		// Wait for the server to be ready
		waitSeconds := c.config.StartWaitSeconds
		if waitSeconds <= 0 {
			waitSeconds = 3
		}

		c.logger.Info("Waiting for server to be ready",
			zap.Int("wait_seconds", waitSeconds),
			zap.String("url", c.config.URL),
		)
		time.Sleep(time.Duration(waitSeconds) * time.Second)
	}

	c.logger.Info("HTTP client ready", zap.String("url", c.config.URL))
	c.running = true
	return nil
}

// Stop terminates the server process (if started) and marks the HTTP client as stopped
func (c *HTTPClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Stopping HTTP client")

	// Kill the server process if we started it
	if c.cmd != nil && c.cmd.Process != nil {
		c.logger.Info("Terminating server process", zap.Int("pid", c.cmd.Process.Pid))
		if err := c.cmd.Process.Kill(); err != nil {
			c.logger.Warn("Failed to kill server process", zap.Error(err))
		}
		_ = c.cmd.Wait()
		c.cmd = nil
	}

	c.running = false
	return nil
}

// IsRunning returns true if the client is ready
func (c *HTTPClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// CallTool calls a tool on the HTTP MCP server
func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("client is not ready")
	}

	c.logger.Debug("Calling tool", zap.String("tool", name))

	// Create JSON-RPC request
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	result, err := c.callHTTPRPC(ctx, "tools/call", params, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tools/call failed: %w", err)
	}

	// Parse result
	var callResult mcp.CallToolResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/call result: %w", err)
	}

	return &callResult, nil
}

// ListTools returns all tools from the HTTP server
func (c *HTTPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("client is not ready")
	}

	c.logger.Debug("Listing tools")

	result, err := c.callHTTPRPC(ctx, "tools/list", nil, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tools/list call failed: %w", err)
	}

	// Parse the result
	var listResult struct {
		Tools []mcp.Tool `json:"tools"`
	}

	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/list result: %w", err)
	}

	c.logger.Info("Listed tools from HTTP server", zap.Int("count", len(listResult.Tools)))
	return listResult.Tools, nil
}

// callHTTPRPC makes a JSON-RPC call over HTTP
func (c *HTTPClient) callHTTPRPC(ctx context.Context, method string, params interface{}, timeout time.Duration) (json.RawMessage, error) {
	// Generate request ID
	id := c.nextID.Add(1)

	// Create JSON-RPC request
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.Debug("Sending HTTP JSON-RPC request",
		zap.Int64("id", id),
		zap.String("method", method),
		zap.String("url", c.config.URL),
	)

	// Create HTTP request with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Decode response
	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// GetName returns the server name
func (c *HTTPClient) GetName() string {
	return c.config.Name
}

// GetType returns the server type
func (c *HTTPClient) GetType() string {
	return c.config.Type
}

// monitorProcess monitors the launched server process
func (c *HTTPClient) monitorProcess() {
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}

	err := c.cmd.Wait()

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()

	if err != nil {
		c.logger.Error("Server process exited with error", zap.Error(err))
	} else {
		c.logger.Info("Server process exited normally")
	}
}
