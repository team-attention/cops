package config

import "github.com/team-attention/cops/shared/domain"

// GlobalConfig represents the ~/.cops/config.json structure.
type GlobalConfig struct {
	Projects []*domain.Project `json:"projects"`
}

// LocalConfig represents the {projectPath}/.cops/config.json structure.
type LocalConfig struct {
	ID domain.ID `json:"id"`
}

// ConfigPort defines the interface for configuration storage.
type ConfigPort interface {
	// LoadGlobalConfig loads the global configuration from ~/.cops/config.json.
	LoadGlobalConfig() (*GlobalConfig, error)

	// SaveGlobalConfig saves the global configuration to ~/.cops/config.json.
	SaveGlobalConfig(cfg *GlobalConfig) error

	// LoadLocalConfig loads the local configuration from {projectPath}/.cops/config.json.
	LoadLocalConfig(projectPath string) (*LocalConfig, error)

	// SaveLocalConfig saves the local configuration to {projectPath}/.cops/config.json.
	SaveLocalConfig(projectPath string, cfg *LocalConfig) error

	// LocalConfigExists checks if a local config exists at the given path.
	LocalConfigExists(projectPath string) bool
}
