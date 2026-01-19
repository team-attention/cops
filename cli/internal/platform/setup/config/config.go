package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	App     AppConfig
	Logging LoggingConfig
	API     APIConfig
	Paths   PathsConfig
	Upgrade UpgradeConfig
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

// APIConfig holds API server settings.
type APIConfig struct {
	URL     string
	Timeout time.Duration
}

// PathsConfig holds path settings.
type PathsConfig struct {
	BaseDir        string // Global configuration directory (~/.cops)
	LocalConfigDir string // Local configuration directory name (.cops)
}

// UpgradeConfig holds auto-upgrade settings.
type UpgradeConfig struct {
	Owner     string
	Repo      string
	AutoCheck bool
}

// LoadConfig loads configuration from environment variables using Viper.
// Environment variables are prefixed with COPS_ (e.g., COPS_LOG_LEVEL).
// Version is read from COPS_APP_VERSION environment variable.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set environment variable prefix
	v.SetEnvPrefix("COPS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("app.name", "cops")
	v.SetDefault("app.version", "dev")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("api.url", "https://cops-api-392947101616.asia-northeast3.run.app")
	v.SetDefault("api.timeout", "30s")
	v.SetDefault("paths.basedir", "~/.cops")
	v.SetDefault("paths.localconfigdir", ".cops")
	v.SetDefault("upgrade.owner", "team-attention")
	v.SetDefault("upgrade.repo", "cops")
	v.SetDefault("upgrade.autocheck", true)

	cfg := &Config{
		App: AppConfig{
			Name:    v.GetString("app.name"),
			Version: v.GetString("app.version"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("logging.level"),
			Format: v.GetString("logging.format"),
		},
		API: APIConfig{
			URL:     v.GetString("api.url"),
			Timeout: v.GetDuration("api.timeout"),
		},
		Paths: PathsConfig{
			BaseDir:        v.GetString("paths.basedir"),
			LocalConfigDir: v.GetString("paths.localconfigdir"),
		},
		Upgrade: UpgradeConfig{
			Owner:     v.GetString("upgrade.owner"),
			Repo:      v.GetString("upgrade.repo"),
			AutoCheck: v.GetBool("upgrade.autocheck"),
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
