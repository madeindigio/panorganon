package server

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
)

// Server represents an MCP server that can run with different transports
type Server interface {
	Start(ctx context.Context) error
	Stop() error
}

// Options contains common server options
type Options struct {
	Config config.ServerConfig
	Logger *zap.Logger

	// Tool handlers will be set by the server implementation
	MCPServer *server.MCPServer
}

// New creates a new server instance based on configuration
func New(cfg config.ServerConfig, logger *zap.Logger, mcpServer *server.MCPServer) ([]Server, error) {
	servers := make([]Server, 0)

	// Create stdio server if enabled
	if cfg.Stdio.Enabled {
		stdioServer := NewStdioServer(logger, mcpServer)
		servers = append(servers, stdioServer)
	}

	// Create HTTP server if enabled
	if cfg.HTTP.Enabled {
		httpServer := NewHTTPServer(cfg.HTTP, logger, mcpServer)
		servers = append(servers, httpServer)
	}

	return servers, nil
}
