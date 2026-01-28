package downstream

import (
	"context"
	"fmt"
	"sync"

	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
)

// Manager manages all downstream MCP server clients
type Manager struct {
	clients map[string]DownstreamClient
	config  []config.DownstreamServer
	logger  *zap.Logger
	mu      sync.RWMutex
}

// NewManager creates a new downstream server manager
func NewManager(cfg []config.DownstreamServer, logger *zap.Logger) *Manager {
	return &Manager{
		clients: make(map[string]DownstreamClient),
		config:  cfg,
		logger:  logger.With(zap.String("component", "downstream-manager")),
	}
}

// Initialize creates all client instances
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Initializing downstream servers", zap.Int("count", len(m.config)))

	for _, serverCfg := range m.config {
		var client DownstreamClient

		switch serverCfg.Type {
		case "stdio":
			client = NewStdioClient(serverCfg, m.logger)
		case "streamable-http":
			client = NewHTTPClient(serverCfg, m.logger)
		case "sse":
			client = NewSSEClient(serverCfg, m.logger)
		default:
			return fmt.Errorf("unknown server type: %s", serverCfg.Type)
		}

		m.clients[serverCfg.Name] = client
		m.logger.Info("Registered downstream server",
			zap.String("name", serverCfg.Name),
			zap.String("type", serverCfg.Type),
		)

		// Start keepalive servers immediately
		if serverCfg.KeepAlive {
			m.logger.Info("Starting keepalive server", zap.String("name", serverCfg.Name))
			if err := client.Start(context.Background()); err != nil {
				m.logger.Error("Failed to start keepalive server",
					zap.String("name", serverCfg.Name),
					zap.Error(err),
				)
				return fmt.Errorf("failed to start keepalive server %s: %w", serverCfg.Name, err)
			}
		}
	}

	return nil
}

// GetOrStart gets a client and starts it if not already running
func (m *Manager) GetOrStart(ctx context.Context, serverName string) (DownstreamClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[serverName]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	if !client.IsRunning() {
		m.logger.Info("Starting on-demand server", zap.String("name", serverName))
		if err := client.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start server %s: %w", serverName, err)
		}
	}

	return client, nil
}

// Get returns a client without starting it
func (m *Manager) Get(serverName string) (DownstreamClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[serverName]
	return client, exists
}

// GetClient returns a client by name (or nil if not found)
func (m *Manager) GetClient(serverName string) DownstreamClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.clients[serverName]
}

// GetServerInfo returns server configuration information
func (m *Manager) GetServerInfo(serverName string) *ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[serverName]
	if !exists {
		return nil
	}

	// Find config for this server
	for _, cfg := range m.config {
		if cfg.Name == serverName {
			return &ServerInfo{
				Name:      serverName,
				Type:      client.GetType(),
				Running:   client.IsRunning(),
				KeepAlive: cfg.KeepAlive,
			}
		}
	}

	return nil
}

// Stop stops a server if it's not keepalive
func (m *Manager) Stop(serverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[serverName]
	if !exists {
		return fmt.Errorf("server not found: %s", serverName)
	}

	// Find the config for this server to check keepalive
	var isKeepAlive bool
	for _, cfg := range m.config {
		if cfg.Name == serverName {
			isKeepAlive = cfg.KeepAlive
			break
		}
	}

	// Don't stop keepalive servers
	if isKeepAlive {
		m.logger.Debug("Not stopping keepalive server", zap.String("name", serverName))
		return nil
	}

	m.logger.Info("Stopping on-demand server", zap.String("name", serverName))
	return client.Stop()
}

// StopAll stops all running servers
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Stopping all downstream servers")

	var errors []error
	for name, client := range m.clients {
		if client.IsRunning() {
			m.logger.Info("Stopping server", zap.String("name", name))
			if err := client.Stop(); err != nil {
				m.logger.Error("Failed to stop server",
					zap.String("name", name),
					zap.Error(err),
				)
				errors = append(errors, fmt.Errorf("failed to stop %s: %w", name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errors)
	}

	return nil
}

// ListServers returns information about all configured servers
func (m *Manager) ListServers() []ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]ServerInfo, 0, len(m.clients))
	for name, client := range m.clients {
		// Find config for this server
		var cfg config.DownstreamServer
		for _, c := range m.config {
			if c.Name == name {
				cfg = c
				break
			}
		}

		servers = append(servers, ServerInfo{
			Name:      name,
			Type:      client.GetType(),
			Running:   client.IsRunning(),
			KeepAlive: cfg.KeepAlive,
		})
	}

	return servers
}

// ServerInfo contains information about a downstream server
type ServerInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Running   bool   `json:"running"`
	KeepAlive bool   `json:"keepalive"`
}
