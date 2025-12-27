package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all API application configuration.
type Config struct {
	App     AppConfig
	HTTP    HTTPConfig
	Logging LoggingConfig
	MongoDB MongoDBConfig
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name    string `env:"APP_NAME" envDefault:"cops-api"`
	Version string `env:"APP_VERSION" envDefault:"0.0.1"`
	Env     string `env:"APP_ENV" envDefault:"development"`
}

// HTTPConfig holds HTTP server and CORS settings.
type HTTPConfig struct {
	Port            int           `env:"PORT" envDefault:"8080"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`

	// CORS settings
	CORSAllowOrigins string `env:"CORS_ALLOW_ORIGINS" envDefault:"*"`
	CORSAllowMethods string `env:"CORS_ALLOW_METHODS" envDefault:"GET,POST,PUT,DELETE,OPTIONS,PATCH"`
	CORSAllowHeaders string `env:"CORS_ALLOW_HEADERS" envDefault:"Origin,Content-Type,Accept,Connect-Protocol-Version"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `env:"LOGGING_LEVEL" envDefault:"info"`
	Format string `env:"LOGGING_FORMAT" envDefault:"json"`
}

// MongoDBConfig holds MongoDB connection settings.
type MongoDBConfig struct {
	URI      string `env:"MONGODB_URI,required"`
	Database string `env:"MONGODB_DATABASE,required"`
}

// InitConfig loads configuration from environment variables.
func InitConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
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

	// Validate port range
	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", cfg.HTTP.Port)
	}

	return nil
}
