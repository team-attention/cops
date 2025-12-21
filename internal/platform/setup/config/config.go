package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	App     AppConfig
	Logging LoggingConfig
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

	cfg := &Config{
		App: AppConfig{
			Name:    v.GetString("app.name"),
			Version: v.GetString("app.version"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("logging.level"),
			Format: v.GetString("logging.format"),
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
