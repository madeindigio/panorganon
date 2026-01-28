package downstream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCClient handles JSON-RPC 2.0 communication over io streams
type JSONRPCClient struct {
	reader        io.Reader
	writer        io.Writer
	logger        *zap.Logger
	nextID        atomic.Int64
	pendingCalls  map[int64]chan *JSONRPCResponse
	pendingMux    sync.RWMutex
	readLoopOnce  sync.Once
	stopCh        chan struct{}
	encoder       *json.Encoder
	decoder       *json.Decoder
}

// NewJSONRPCClient creates a new JSON-RPC client
func NewJSONRPCClient(reader io.Reader, writer io.Writer, logger *zap.Logger) *JSONRPCClient {
	return &JSONRPCClient{
		reader:       reader,
		writer:       writer,
		logger:       logger.With(zap.String("component", "jsonrpc")),
		pendingCalls: make(map[int64]chan *JSONRPCResponse),
		stopCh:       make(chan struct{}),
		encoder:      json.NewEncoder(writer),
		decoder:      json.NewDecoder(bufio.NewReader(reader)),
	}
}

// Start starts the read loop
func (c *JSONRPCClient) Start() {
	c.readLoopOnce.Do(func() {
		go c.readLoop()
	})
}

// Stop stops the client
func (c *JSONRPCClient) Stop() {
	close(c.stopCh)
}

// readLoop continuously reads responses from the server
func (c *JSONRPCClient) readLoop() {
	c.logger.Debug("Starting JSON-RPC read loop")

	for {
		select {
		case <-c.stopCh:
			c.logger.Debug("Stopping JSON-RPC read loop")
			return
		default:
			var resp JSONRPCResponse
			if err := c.decoder.Decode(&resp); err != nil {
				if err == io.EOF {
					c.logger.Debug("EOF received, stopping read loop")
					return
				}
				c.logger.Error("Failed to decode JSON-RPC response", zap.Error(err))
				continue
			}

			c.logger.Debug("Received JSON-RPC response", zap.Int64("id", resp.ID))

			// Find the pending call
			c.pendingMux.RLock()
			ch, exists := c.pendingCalls[resp.ID]
			c.pendingMux.RUnlock()

			if exists {
				select {
				case ch <- &resp:
				case <-time.After(1 * time.Second):
					c.logger.Warn("Timeout sending response to caller", zap.Int64("id", resp.ID))
				}
			} else {
				c.logger.Warn("Received response for unknown request ID", zap.Int64("id", resp.ID))
			}
		}
	}
}

// Call makes a JSON-RPC call with timeout
func (c *JSONRPCClient) Call(ctx context.Context, method string, params interface{}, timeout time.Duration) (json.RawMessage, error) {
	// Generate request ID
	id := c.nextID.Add(1)

	// Create response channel
	respCh := make(chan *JSONRPCResponse, 1)

	// Register pending call
	c.pendingMux.Lock()
	c.pendingCalls[id] = respCh
	c.pendingMux.Unlock()

	// Ensure cleanup
	defer func() {
		c.pendingMux.Lock()
		delete(c.pendingCalls, id)
		c.pendingMux.Unlock()
		close(respCh)
	}()

	// Create and send request
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	c.logger.Debug("Sending JSON-RPC request",
		zap.Int64("id", id),
		zap.String("method", method),
	)

	if err := c.encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Wait for response with timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("request timeout after %s", timeout)
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// ListToolsRPC calls the tools/list method
func (c *JSONRPCClient) ListToolsRPC(ctx context.Context) ([]mcp.Tool, error) {
	result, err := c.Call(ctx, "tools/list", nil, 30*time.Second)
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

	c.logger.Info("Listed tools", zap.Int("count", len(listResult.Tools)))
	return listResult.Tools, nil
}

// CallToolRPC calls the tools/call method
func (c *JSONRPCClient) CallToolRPC(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	result, err := c.Call(ctx, "tools/call", params, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tools/call failed: %w", err)
	}

	// Parse the result
	var callResult mcp.CallToolResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/call result: %w", err)
	}

	return &callResult, nil
}
