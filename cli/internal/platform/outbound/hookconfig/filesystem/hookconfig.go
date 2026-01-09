package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/team-attention/cops/cli/internal/platform/outbound/hookconfig"
)

const (
	claudeSettingsDir  = ".claude"
	claudeSettingsFile = "settings.json"
	copsConfigDir      = ".cops"
	authConfigFile     = "auth.json"
)

// FilesystemHookConfig implements HookConfigPort using filesystem storage.
type FilesystemHookConfig struct {
	logger  *slog.Logger
	homeDir string
}

// NewFilesystemHookConfig creates a new filesystem-based Hook config adapter.
func NewFilesystemHookConfig(l *slog.Logger) hookconfig.HookConfigPort {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home directory cannot be determined
		homeDir = "."
	}

	return &FilesystemHookConfig{
		logger:  l.With(slog.String("name", "platform.hookconfig.filesystem")),
		homeDir: homeDir,
	}
}

// LoadConfig loads and merges Hook configuration from project and global sources.
func (f *FilesystemHookConfig) LoadConfig(ctx context.Context, projectDir string) (*hookconfig.Config, error) {
	// 1. Load Hook settings from project
	hookSettings, err := f.LoadHookSettings(ctx, projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load hook settings: %w", err)
	}

	// 2. Load Auth config from global
	authConfig, err := f.LoadAuthConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth config: %w", err)
	}

	// 3. Merge and create Config
	config := &hookconfig.Config{
		Hook: hookSettings,
		Auth: authConfig,
	}

	// 4. Validate merged config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	f.logger.Debug("loaded hook configuration",
		slog.Bool("hookEnabled", config.IsEnabled()),
		slog.Bool("hasAuth", authConfig != nil),
	)

	return config, nil
}

// LoadHookSettings loads Hook settings from .claude/settings.json.
func (f *FilesystemHookConfig) LoadHookSettings(ctx context.Context, projectDir string) (*hookconfig.HookSettings, error) {
	settingsPath := filepath.Join(projectDir, claudeSettingsDir, claudeSettingsFile)

	// 1. Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		f.logger.Debug("claude settings file not found",
			slog.String("path", settingsPath),
		)
		return nil, nil
	}

	// 2. Read file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	// 3. Parse JSON
	var settings hookconfig.ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	// 4. Return cops section (may be nil)
	return settings.Cops, nil
}

// LoadAuthConfig loads API key configuration from ~/.cops/auth.json.
func (f *FilesystemHookConfig) LoadAuthConfig(ctx context.Context) (*hookconfig.AuthConfig, error) {
	authPath := filepath.Join(f.homeDir, copsConfigDir, authConfigFile)

	// 1. Check if file exists
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		f.logger.Debug("auth config file not found",
			slog.String("path", authPath),
		)
		return nil, nil
	}

	// 2. Read file
	data, err := os.ReadFile(authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth config: %w", err)
	}

	// 3. Parse JSON
	var authConfig hookconfig.AuthConfig
	if err := json.Unmarshal(data, &authConfig); err != nil {
		return nil, fmt.Errorf("failed to parse auth config: %w", err)
	}

	return &authConfig, nil
}

// Compile-time interface verification
var _ hookconfig.HookConfigPort = (*FilesystemHookConfig)(nil)
