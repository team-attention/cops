package hookconfig

import "errors"

var (
	// ErrAPIKeyRequired is returned when Hook is enabled but API key is missing.
	ErrAPIKeyRequired = errors.New("API key is required when Hook is enabled")

	// ErrInvalidConfig is returned when configuration parsing fails.
	ErrInvalidConfig = errors.New("invalid configuration format")

	// ErrConfigNotFound is returned when a required config file is not found.
	ErrConfigNotFound = errors.New("configuration file not found")
)
