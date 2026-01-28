package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sevir/panorganon/internal/database"
	"github.com/sevir/panorganon/internal/downstream"
	"go.uber.org/zap"
)

// DiscoveryMetrics holds metrics for tool discovery operations
type DiscoveryMetrics struct {
	TotalToolsDiscovered int
	FailedDiscoveries    int
	LastDiscoveryTime    time.Time
	DiscoveryDuration    time.Duration
}

// DiscoveryService handles tool discovery from downstream servers
type DiscoveryService struct {
	manager         *downstream.Manager
	db              *database.DB
	logger          *zap.Logger
	refreshInterval time.Duration
	stopCh          chan struct{}
	metrics         DiscoveryMetrics
	metricsMux      sync.RWMutex
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(manager *downstream.Manager, db *database.DB, logger *zap.Logger, refreshInterval time.Duration) *DiscoveryService {
	if refreshInterval == 0 {
		refreshInterval = 1 * time.Hour // Default 1 hour
	}

	return &DiscoveryService{
		manager:         manager,
		db:              db,
		logger:          logger.With(zap.String("component", "discovery")),
		refreshInterval: refreshInterval,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the periodic discovery process
func (ds *DiscoveryService) Start(ctx context.Context) error {
	ds.logger.Info("Starting discovery service", zap.Duration("refresh_interval", ds.refreshInterval))

	// Initial discovery on startup
	if err := ds.DiscoverAll(ctx); err != nil {
		ds.logger.Warn("Initial discovery failed", zap.Error(err))
	}

	// Start periodic refresh
	go ds.periodicRefresh(ctx)

	return nil
}

// Stop stops the periodic discovery process
func (ds *DiscoveryService) Stop() {
	ds.logger.Info("Stopping discovery service")
	close(ds.stopCh)
}

// periodicRefresh runs discovery at regular intervals
func (ds *DiscoveryService) periodicRefresh(ctx context.Context) {
	ticker := time.NewTicker(ds.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopCh:
			return
		case <-ticker.C:
			ds.logger.Info("Running periodic tool discovery")
			if err := ds.DiscoverAll(ctx); err != nil {
				ds.logger.Error("Periodic discovery failed", zap.Error(err))
			}
		}
	}
}

// DiscoverAll discovers tools from all configured downstream servers
func (ds *DiscoveryService) DiscoverAll(ctx context.Context) error {
	startTime := time.Now()
	ds.logger.Info("Starting tool discovery for all servers")

	serverInfos := ds.manager.ListServers()
	totalTools := 0
	failedServers := 0

	for _, info := range serverInfos {
		ds.logger.Info("Discovering tools from server", zap.String("server", info.Name))

		count, err := ds.DiscoverServer(ctx, info.Name)
		if err != nil {
			ds.logger.Error("Failed to discover tools from server",
				zap.String("server", info.Name),
				zap.Error(err),
			)
			failedServers++
			continue
		}

		totalTools += count
		ds.logger.Info("Discovered tools from server",
			zap.String("server", info.Name),
			zap.Int("count", count),
		)
	}

	duration := time.Since(startTime)

	// Update metrics
	ds.metricsMux.Lock()
	ds.metrics.TotalToolsDiscovered = totalTools
	ds.metrics.FailedDiscoveries = failedServers
	ds.metrics.LastDiscoveryTime = startTime
	ds.metrics.DiscoveryDuration = duration
	ds.metricsMux.Unlock()

	ds.logger.Info("Tool discovery completed",
		zap.Int("total_tools", totalTools),
		zap.Int("failed_servers", failedServers),
		zap.Duration("duration", duration),
	)

	if failedServers == len(serverInfos) && len(serverInfos) > 0 {
		return fmt.Errorf("all %d servers failed to discover tools", failedServers)
	}

	return nil
}

// DiscoverServer discovers tools from a specific downstream server
func (ds *DiscoveryService) DiscoverServer(ctx context.Context, serverName string) (int, error) {
	ds.logger.Debug("Starting discovery for server", zap.String("server", serverName))

	// Get the client from manager
	client := ds.manager.GetClient(serverName)
	if client == nil {
		return 0, fmt.Errorf("server not found: %s", serverName)
	}

	wasRunning := client.IsRunning()

	// Start the server if it's not running
	if !wasRunning {
		ds.logger.Debug("Starting server for discovery", zap.String("server", serverName))
		if err := client.Start(ctx); err != nil {
			return 0, fmt.Errorf("failed to start server %s: %w", serverName, err)
		}

		// Give the server a moment to initialize
		time.Sleep(500 * time.Millisecond)
	}

	// Ensure we stop the server if it wasn't running before (and it's not keepalive)
	if !wasRunning {
		defer func() {
			// Get the server config to check keepalive
			serverInfo := ds.manager.GetServerInfo(serverName)
			if serverInfo != nil && !serverInfo.KeepAlive {
				ds.logger.Debug("Stopping non-keepalive server after discovery", zap.String("server", serverName))
				if err := client.Stop(); err != nil {
					ds.logger.Warn("Failed to stop server after discovery",
						zap.String("server", serverName),
						zap.Error(err),
					)
				}
			}
		}()
	}

	// List tools from the server
	tools, err := client.ListTools(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list tools from %s: %w", serverName, err)
	}

	// Save server record
	serverRecord := database.ServerRecord{
		ID:       serverName,
		Name:     serverName,
		Type:     client.GetType(),
		Config:   "", // Could serialize config here if needed
		Status:   "active",
		LastSeen: time.Now(),
	}

	if err := ds.db.SaveServer(serverRecord); err != nil {
		return 0, fmt.Errorf("failed to save server record: %w", err)
	}

	// Delete old tools for this server (to handle removed tools)
	if err := ds.db.DeleteToolsByServer(serverName); err != nil {
		ds.logger.Warn("Failed to delete old tools", zap.String("server", serverName), zap.Error(err))
	}

	// Save tools to database
	if err := ds.db.SaveToolsFromMCP(serverName, serverName, tools); err != nil {
		return 0, fmt.Errorf("failed to save tools to database: %w", err)
	}

	ds.logger.Info("Successfully discovered tools",
		zap.String("server", serverName),
		zap.Int("tool_count", len(tools)),
	)

	return len(tools), nil
}

// GetMetrics returns the current discovery metrics
func (ds *DiscoveryService) GetMetrics() DiscoveryMetrics {
	ds.metricsMux.RLock()
	defer ds.metricsMux.RUnlock()
	return ds.metrics
}
