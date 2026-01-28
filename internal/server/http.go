package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
)

// HTTPServer implements Server interface for streamable-http transport
type HTTPServer struct {
	config           config.HTTPConfig
	logger           *zap.Logger
	streamableServer *server.StreamableHTTPServer
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(cfg config.HTTPConfig, logger *zap.Logger, mcpServer *server.MCPServer) *HTTPServer {
	addr := fmt.Sprintf(":%d", cfg.Port)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         addr,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create streamable HTTP server with mcp-go
	streamableServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithEndpointPath(cfg.Endpoint),
		server.WithStreamableHTTPServer(httpServer),
	)

	return &HTTPServer{
		config:           cfg,
		logger:           logger.With(zap.String("transport", "streamable-http")),
		streamableServer: streamableServer,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	s.logger.Info("Starting streamable HTTP server",
		zap.String("address", addr),
		zap.String("endpoint", s.config.Endpoint),
	)

	// Start server - this blocks
	if err := s.streamableServer.Start(addr); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Streamable HTTP server error", zap.Error(err))
		return err
	}

	return nil
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	s.logger.Info("Stopping streamable HTTP server")

	if s.streamableServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.streamableServer.Shutdown(ctx); err != nil {
		s.logger.Error("Error shutting down streamable HTTP server", zap.Error(err))
		return err
	}

	return nil
}
