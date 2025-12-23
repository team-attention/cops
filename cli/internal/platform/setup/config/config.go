package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	App       AppConfig
	Logging   LoggingConfig
	Collector CollectorConfig
	API       APIConfig
	Daemon    DaemonConfig
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name    string
	Version string
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string
	Format string
}

// CollectorConfig holds collector server settings.
type CollectorConfig struct {
	URL     string
	Timeout time.Duration
}

// APIConfig holds API server settings.
type APIConfig struct {
	URL     string
	Timeout time.Duration
}

// DaemonConfig holds daemon settings.
type DaemonConfig struct {
	BinaryPath string
}

// LoadConfig loads configuration from environment variables using Viper.
// Environment variables are prefixed with COPS_ (e.g., COPS_LOG_LEVEL).
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set environment variable prefix
	v.SetEnvPrefix("COPS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("app.name", "cops")
	v.SetDefault("app.version", "0.0.1")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("collector.url", "http://localhost:8080")
	v.SetDefault("collector.timeout", "30s")
	v.SetDefault("api.url", "http://localhost:8081")
	v.SetDefault("api.timeout", "30s")
	v.SetDefault("daemon.binarypath", "~/.cops/bin/cops-daemon")

	cfg := &Config{
		App: AppConfig{
			Name:    v.GetString("app.name"),
			Version: v.GetString("app.version"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("logging.level"),
			Format: v.GetString("logging.format"),
		},
		Collector: CollectorConfig{
			URL:     v.GetString("collector.url"),
			Timeout: v.GetDuration("collector.timeout"),
		},
		API: APIConfig{
			URL:     v.GetString("api.url"),
			Timeout: v.GetDuration("api.timeout"),
		},
		Daemon: DaemonConfig{
			BinaryPath: v.GetString("daemon.binarypath"),
		},
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validateConfig validates the configuration values.
func validateConfig(cfg *Config) error {
	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(cfg.Logging.Level)] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", cfg.Logging.Level)
	}

	// Validate log format
	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[strings.ToLower(cfg.Logging.Format)] {
		return fmt.Errorf("invalid log format: %s (must be text or json)", cfg.Logging.Format)
	}

	return nil
}
