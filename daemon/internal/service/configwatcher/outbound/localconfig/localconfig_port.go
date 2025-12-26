package localconfig

import (
	"github.com/team-attention/cops/shared/domain"
)

// LocalConfig represents the {projectPath}/.cops/config.json structure.
type LocalConfig struct {
	ID domain.ID `json:"id"`
}

// LocalConfigPort defines the interface for reading local project configs.
type LocalConfigPort interface {
	// LoadLocalConfig loads the local configuration from {projectPath}/.cops/config.json.
	// Returns nil, error if file does not exist or cannot be parsed.
	LoadLocalConfig(projectPath string) (*LocalConfig, error)
}
