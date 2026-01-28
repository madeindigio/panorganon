package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
server:
  stdio:
    enabled: true
  http:
    enabled: false
    port: 8080
    endpoint: /mcp

logging:
  level: info
  file: ./logs/test.log

database:
  path: ./test.db

sampling:
  provider: anthropic
  api_key: test-key
  model: claude-3-5-sonnet-20241022

downstream_servers:
  - name: test-server
    type: stdio
    command: test-command
    args:
      - arg1
      - arg2
    keepalive: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify server config
	if !cfg.Server.Stdio.Enabled {
		t.Error("Expected stdio to be enabled")
	}

	if cfg.Server.HTTP.Enabled {
		t.Error("Expected HTTP to be disabled")
	}

	if cfg.Server.HTTP.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.HTTP.Port)
	}

	// Verify logging config
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", cfg.Logging.Level)
	}

	// Verify database config
	if cfg.Database.Path != "./test.db" {
		t.Errorf("Expected database path './test.db', got '%s'", cfg.Database.Path)
	}

	// Verify sampling config
	if cfg.Sampling.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", cfg.Sampling.Provider)
	}

	if cfg.Sampling.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", cfg.Sampling.Model)
	}

	// Verify downstream servers
	if len(cfg.DownstreamServers) != 1 {
		t.Fatalf("Expected 1 downstream server, got %d", len(cfg.DownstreamServers))
	}

	server := cfg.DownstreamServers[0]
	if server.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", server.Name)
	}

	if server.Type != "stdio" {
		t.Errorf("Expected server type 'stdio', got '%s'", server.Type)
	}

	if !server.KeepAlive {
		t.Error("Expected server to have keepalive enabled")
	}

	if len(server.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(server.Args))
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Stdio: StdioConfig{Enabled: true},
				},
				Logging: LoggingConfig{
					Level: "info",
					File:  "./test.log",
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
				Sampling: SamplingConfig{
					Provider: "anthropic",
					APIKey:   "test-key",
					Model:    "test-model",
				},
				DownstreamServers: []DownstreamServer{
					{
						Name:    "test",
						Type:    "stdio",
						Command: "test-cmd",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "no transport enabled",
			config: Config{
				Server: ServerConfig{
					Stdio: StdioConfig{Enabled: false},
					HTTP:  HTTPConfig{Enabled: false},
				},
				Logging: LoggingConfig{Level: "info", File: "./test.log"},
				Database: DatabaseConfig{Path: "./test.db"},
				Sampling: SamplingConfig{Provider: "anthropic", APIKey: "key", Model: "model"},
			},
			expectErr: true,
		},
		{
			name: "invalid HTTP port",
			config: Config{
				Server: ServerConfig{
					Stdio: StdioConfig{Enabled: true},
					HTTP:  HTTPConfig{Enabled: true, Port: 99999},
				},
				Logging: LoggingConfig{Level: "info", File: "./test.log"},
				Database: DatabaseConfig{Path: "./test.db"},
				Sampling: SamplingConfig{Provider: "anthropic", APIKey: "key", Model: "model"},
			},
			expectErr: true,
		},
		{
			name: "missing sampling provider",
			config: Config{
				Server: ServerConfig{
					Stdio: StdioConfig{Enabled: true},
				},
				Logging: LoggingConfig{Level: "info", File: "./test.log"},
				Database: DatabaseConfig{Path: "./test.db"},
				Sampling: SamplingConfig{
					APIKey: "key",
					Model:  "model",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error, got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestEnvironmentVariableExpansion(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_API_KEY", "secret-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	// Create temporary config file with env var
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
server:
  stdio:
    enabled: true

logging:
  level: info
  file: ./test.log

database:
  path: ./test.db

sampling:
  provider: anthropic
  api_key: ${TEST_API_KEY}
  model: test-model

downstream_servers: []
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config and verify env var was expanded
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Sampling.APIKey != "secret-key-123" {
		t.Errorf("Expected API key 'secret-key-123', got '%s'", cfg.Sampling.APIKey)
	}
}
