package filesystem

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig"
)

const (
	configFileName = "config.json"
)

// FilesystemLocalConfigAdapter implements LocalConfigPort using filesystem.
type FilesystemLocalConfigAdapter struct {
	logger         *slog.Logger
	localConfigDir string
}

// NewFilesystemLocalConfigAdapter creates a new filesystem local config adapter.
func NewFilesystemLocalConfigAdapter(l *slog.Logger, localConfigDir string) *FilesystemLocalConfigAdapter {
	return &FilesystemLocalConfigAdapter{
		logger:         l.With(slog.String("name", "configwatcher.localconfig.filesystem")),
		localConfigDir: localConfigDir,
	}
}

// LoadLocalConfig loads the local configuration from {projectPath}/{localConfigDir}/config.json.
func (a *FilesystemLocalConfigAdapter) LoadLocalConfig(projectPath string) (*localconfig.LocalConfig, error) {
	configPath := filepath.Join(projectPath, a.localConfigDir, configFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg localconfig.LocalConfig
	if err := sonic.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Compile-time interface verification
var _ localconfig.LocalConfigPort = (*FilesystemLocalConfigAdapter)(nil)
