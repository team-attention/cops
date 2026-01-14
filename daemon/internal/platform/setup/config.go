package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all daemon application configuration.
type Config struct {
	App     AppConfig
	Logging LoggingConfig
	API     APIConfig
	Cops    CopsConfig
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name    string `env:"COPS_APP_NAME" envDefault:"cops-daemon"`
	Version string `env:"COPS_APP_VERSION" envDefault:"0.0.1"`
	Env     string `env:"COPS_APP_ENV" envDefault:"development"`
	Debug   bool   `env:"COPS_DEBUG" envDefault:"false"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `env:"COPS_LOG_LEVEL" envDefault:"info"`
	Format string `env:"COPS_LOG_FORMAT" envDefault:"text"`
}

// APIConfig holds API server connection settings.
type APIConfig struct {
	URL     string        `env:"COPS_API_URL" envDefault:"https://cops-api-392947101616.asia-northeast3.run.app"`
	Timeout time.Duration `env:"COPS_API_TIMEOUT" envDefault:"30s"`
}

// CopsConfig holds COps-specific settings.
type CopsConfig struct {
	GlobalConfigPath string        `env:"COPS_GLOBAL_CONFIG_PATH" envDefault:"~/.cops/config.json"`
	DaemonDataDir    string        `env:"COPS_DAEMON_DATA_DIR" envDefault:"~/.cops/daemon"`
	FlushInterval    time.Duration `env:"COPS_FLUSH_INTERVAL" envDefault:"15s"`
	MaxBatchSize     int           `env:"COPS_MAX_BATCH_SIZE" envDefault:"100"`
	SocketPath       string        `env:"COPS_SOCKET_PATH" envDefault:"~/.cops/daemon.sock"`
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand home directory in paths
	cfg.Cops.GlobalConfigPath = expandPath(cfg.Cops.GlobalConfigPath)
	cfg.Cops.DaemonDataDir = expandPath(cfg.Cops.DaemonDataDir)

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// expandPath expands ~ to the home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
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
	if !validLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", cfg.Logging.Level)
	}

	// Validate log format
	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[cfg.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (must be text or json)", cfg.Logging.Format)
	}

	// Validate max batch size
	if cfg.Cops.MaxBatchSize < 1 {
		return fmt.Errorf("max batch size must be at least 1")
	}

	return nil
}
