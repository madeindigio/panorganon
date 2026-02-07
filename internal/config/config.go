package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete Panorganon configuration
type Config struct {
	Server             ServerConfig         `mapstructure:"server"`
	Logging            LoggingConfig        `mapstructure:"logging"`
	Database           DatabaseConfig       `mapstructure:"database"`
	Sampling           SamplingConfig       `mapstructure:"sampling"`
	Filters            FiltersConfig        `mapstructure:"filters"`
	DownstreamServers  []DownstreamServer   `mapstructure:"downstream_servers"`
}

// ServerConfig defines the MCP server transport configurations
type ServerConfig struct {
	Stdio StdioConfig `mapstructure:"stdio"`
	HTTP  HTTPConfig  `mapstructure:"http"`
}

// StdioConfig for stdio transport
type StdioConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// HTTPConfig for streamable-http transport
type HTTPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Port     int    `mapstructure:"port"`
	Endpoint string `mapstructure:"endpoint"`
}

// LoggingConfig defines logging parameters
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// DatabaseConfig defines database parameters
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// SamplingConfig defines LLM sampling parameters for tool search
type SamplingConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
}

// FiltersConfig defines Lua filter parameters
type FiltersConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	ScriptPath      string        `mapstructure:"script_path"`
	Timeout         time.Duration `mapstructure:"timeout"`
	StrictMode      bool          `mapstructure:"strict_mode"`
	HotReload       bool          `mapstructure:"hot_reload"`
	LogFilteredData bool          `mapstructure:"log_filtered_data"`
}

// DownstreamServer represents a downstream MCP server configuration
type DownstreamServer struct {
	Name             string            `mapstructure:"name"`
	Type             string            `mapstructure:"type"` // stdio, streamable-http, sse
	Command          string            `mapstructure:"command"`
	Args             []string          `mapstructure:"args"`
	URL              string            `mapstructure:"url"`
	Env              map[string]string `mapstructure:"env"`
	KeepAlive        bool              `mapstructure:"keepalive"`
	StartCmd         string            `mapstructure:"start_cmd"`
	StartArgs        []string          `mapstructure:"start_args"`
	StartWaitSeconds int               `mapstructure:"start_wait_seconds"`
}

// LoadConfig loads and parses the configuration file
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Enable environment variable expansion
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand environment variables in sensitive fields
	cfg.Sampling.APIKey = os.ExpandEnv(cfg.Sampling.APIKey)
	cfg.Filters.ScriptPath = os.ExpandEnv(cfg.Filters.ScriptPath)
	for i := range cfg.DownstreamServers {
		for k, v := range cfg.DownstreamServers[i].Env {
			cfg.DownstreamServers[i].Env[k] = os.ExpandEnv(v)
		}
	}

	// Set default values for filters if not specified
	if cfg.Filters.Timeout == 0 {
		cfg.Filters.Timeout = 5 * time.Second
	}

	// Set default start_wait_seconds for servers with start_cmd
	for i := range cfg.DownstreamServers {
		if cfg.DownstreamServers[i].StartCmd != "" && cfg.DownstreamServers[i].StartWaitSeconds <= 0 {
			cfg.DownstreamServers[i].StartWaitSeconds = 3
		}
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate at least one transport is enabled
	if !c.Server.Stdio.Enabled && !c.Server.HTTP.Enabled {
		return fmt.Errorf("at least one transport (stdio or http) must be enabled")
	}

	// Validate HTTP config if enabled
	if c.Server.HTTP.Enabled {
		if c.Server.HTTP.Port <= 0 || c.Server.HTTP.Port > 65535 {
			return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTP.Port)
		}
		if c.Server.HTTP.Endpoint == "" {
			return fmt.Errorf("HTTP endpoint cannot be empty")
		}
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	// Validate database path
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Validate sampling config
	if c.Sampling.Provider != "anthropic" && c.Sampling.Provider != "openai" {
		return fmt.Errorf("invalid sampling provider: %s (must be anthropic or openai)", c.Sampling.Provider)
	}
	if c.Sampling.APIKey == "" {
		return fmt.Errorf("sampling API key cannot be empty")
	}
	if c.Sampling.Model == "" {
		return fmt.Errorf("sampling model cannot be empty")
	}

	// Validate filters config
	if c.Filters.Enabled {
		if c.Filters.ScriptPath == "" {
			return fmt.Errorf("filters enabled but script_path is empty")
		}
		if c.Filters.Timeout <= 0 {
			return fmt.Errorf("filters timeout must be positive")
		}
		// Check if script file exists (optional warning, not error)
		if _, err := os.Stat(c.Filters.ScriptPath); os.IsNotExist(err) {
			// Note: In strict mode, this would be an error, but we allow starting without the file
			// The FilterManager will handle this based on strict_mode setting
		}
	}

	// Validate downstream servers
	if len(c.DownstreamServers) == 0 {
		return fmt.Errorf("at least one downstream server must be configured")
	}

	validTypes := map[string]bool{"stdio": true, "streamable-http": true, "sse": true}
	for i, srv := range c.DownstreamServers {
		if srv.Name == "" {
			return fmt.Errorf("downstream server #%d: name cannot be empty", i)
		}
		if !validTypes[srv.Type] {
			return fmt.Errorf("downstream server '%s': invalid type '%s' (must be stdio, streamable-http, or sse)", srv.Name, srv.Type)
		}
		if srv.Type == "stdio" && srv.Command == "" {
			return fmt.Errorf("downstream server '%s': command cannot be empty for stdio type", srv.Name)
		}
		if (srv.Type == "streamable-http" || srv.Type == "sse") && srv.URL == "" {
			return fmt.Errorf("downstream server '%s': URL cannot be empty for %s type", srv.Name, srv.Type)
		}
		if srv.StartCmd != "" && srv.Type == "stdio" {
			return fmt.Errorf("downstream server '%s': start_cmd is not supported for stdio type (use command instead)", srv.Name)
		}
	}

	return nil
}
