package server

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// StdioServer implements Server interface for stdio transport
type StdioServer struct {
	logger    *zap.Logger
	mcpServer *server.MCPServer
	cancel    context.CancelFunc
}

// NewStdioServer creates a new stdio server
func NewStdioServer(logger *zap.Logger, mcpServer *server.MCPServer) *StdioServer {
	return &StdioServer{
		logger:    logger.With(zap.String("transport", "stdio")),
		mcpServer: mcpServer,
	}
}

// Start starts the stdio server
func (s *StdioServer) Start(ctx context.Context) error {
	s.logger.Info("Starting stdio server")

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start stdio transport - this blocks until stdin is closed
	err := server.ServeStdio(s.mcpServer)
	if err != nil {
		s.logger.Error("Stdio server error", zap.Error(err))
		return err
	}

	return nil
}

// Stop stops the stdio server
func (s *StdioServer) Stop() error {
	s.logger.Info("Stopping stdio server")
	if s.cancel != nil {
		s.cancel()
	}
	// Close stdin to stop the stdio transport
	os.Stdin.Close()
	return nil
}
