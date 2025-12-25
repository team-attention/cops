package filesystem

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
	"github.com/team-attention/cops/shared/domain"
)

const (
	globalConfigFileName = "config.json"
	localConfigDirName   = ".cops"
	localConfigFileName  = "config.json"
)

// FilesystemConfigAdapter implements ConfigPort using filesystem-based JSON storage.
type FilesystemConfigAdapter struct {
	logger           *slog.Logger
	globalConfigPath string
	mu               sync.RWMutex
}

// NewFilesystemConfigAdapter creates a new filesystem config adapter.
func NewFilesystemConfigAdapter(l *slog.Logger) (*FilesystemConfigAdapter, error) {
	logger := l.With(slog.String("name", "tracking.config.filesystem"))

	copsDir, err := pathutil.DefaultCopsConfigDir()
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := pathutil.EnsureDir(copsDir); err != nil {
		return nil, err
	}

	return &FilesystemConfigAdapter{
		logger:           logger,
		globalConfigPath: filepath.Join(copsDir, globalConfigFileName),
	}, nil
}

// LoadGlobalConfig loads the global configuration from ~/.cops/config.json.
func (a *FilesystemConfigAdapter) LoadGlobalConfig() (*config.GlobalConfig, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	data, err := os.ReadFile(a.globalConfigPath)
	if os.IsNotExist(err) {
		// Create empty config file if it doesn't exist
		emptyConfig := &config.GlobalConfig{Projects: []*domain.Project{}}
		a.mu.RUnlock()
		if err := a.SaveGlobalConfig(emptyConfig); err != nil {
			a.mu.RLock()
			return nil, err
		}
		a.mu.RLock()
		return emptyConfig, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg config.GlobalConfig
	if err := sonic.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Projects == nil {
		cfg.Projects = []*domain.Project{}
	}

	return &cfg, nil
}

// SaveGlobalConfig saves the global configuration to ~/.cops/config.json.
func (a *FilesystemConfigAdapter) SaveGlobalConfig(cfg *config.GlobalConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := sonic.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.globalConfigPath, data, 0644)
}

// LoadLocalConfig loads the local configuration from {projectPath}/.cops/config.json.
func (a *FilesystemConfigAdapter) LoadLocalConfig(projectPath string) (*config.LocalConfig, error) {
	configPath := filepath.Join(projectPath, localConfigDirName, localConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg config.LocalConfig
	if err := sonic.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveLocalConfig saves the local configuration to {projectPath}/.cops/config.json.
func (a *FilesystemConfigAdapter) SaveLocalConfig(projectPath string, cfg *config.LocalConfig) error {
	configDir := filepath.Join(projectPath, localConfigDirName)
	if err := pathutil.EnsureDir(configDir); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, localConfigFileName)

	data, err := sonic.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// LocalConfigExists checks if a local config exists at the given path.
func (a *FilesystemConfigAdapter) LocalConfigExists(projectPath string) bool {
	configPath := filepath.Join(projectPath, localConfigDirName, localConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}

// Compile-time interface verification
var _ config.ConfigPort = (*FilesystemConfigAdapter)(nil)
