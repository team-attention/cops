# Implementation Plan: TA-138 Hook Configuration Structure Design

## Overview

This plan describes the implementation of the Hook configuration structure for the C-Ops CLI. The configuration system will support:
1. Project-level Hook settings in `.claude/settings.json`
2. Global authentication in `~/.cops/auth.json`
3. Merging project and global configurations
4. Event type filtering

## Architecture

### Directory Structure

Following the hexagonal architecture pattern established in the codebase:

```
shared/
└── hookconfig/
    ├── hookconfig.go              # Domain models (HookConfig, AuthConfig, EventType)
    ├── hookconfig_test.go         # Unit tests
    └── loader/
        ├── loader_port.go         # HookConfigLoaderPort interface
        └── filesystem/
            ├── filesystem_loader.go       # Filesystem-based loader implementation
            └── filesystem_loader_test.go  # Unit tests
```

### Configuration Files

#### `.claude/settings.json` (Project-level)

The existing `.claude/settings.json` already contains Claude Code settings. We will add a `hooks` section:

```json
{
  "permissions": { ... },
  "enableAllProjectMcpServers": true,
  "hooks": {
    "enabled": true,
    "apiEndpoint": "https://api.cops.example.com",
    "events": {
      "PostToolUse": true,
      "Notification": true,
      "UserPromptSubmit": true,
      "Stop": true,
      "SubagentStop": true,
      "SessionStart": true,
      "SessionEnd": true
    }
  }
}
```

#### `~/.cops/auth.json` (Global)

Extend the existing auth.json structure (currently contains OAuth tokens) to include Hook API key:

```json
{
  "tokens": {
    "accessToken": "...",
    "refreshToken": "...",
    "expiresAt": 1234567890
  },
  "hookApiKey": "cops_hook_xxxxxxxxxxxx"
}
```

## Implementation Tasks

### Task 1: Define Domain Models

**File**: `shared/hookconfig/hookconfig.go`

```go
package hookconfig

// EventType represents a Claude Code Hook event type.
type EventType string

const (
    EventTypePostToolUse       EventType = "PostToolUse"
    EventTypeNotification      EventType = "Notification"
    EventTypeUserPromptSubmit  EventType = "UserPromptSubmit"
    EventTypeStop              EventType = "Stop"
    EventTypeSubagentStop      EventType = "SubagentStop"
    EventTypeSessionStart      EventType = "SessionStart"
    EventTypeSessionEnd        EventType = "SessionEnd"
)

// AllEventTypes returns all valid event types.
func AllEventTypes() []EventType {
    return []EventType{
        EventTypePostToolUse,
        EventTypeNotification,
        EventTypeUserPromptSubmit,
        EventTypeStop,
        EventTypeSubagentStop,
        EventTypeSessionStart,
        EventTypeSessionEnd,
    }
}

// HookSettings contains Hook-specific settings from .claude/settings.json.
type HookSettings struct {
    Enabled     bool              `json:"enabled"`
    APIEndpoint string            `json:"apiEndpoint"`
    Events      map[EventType]bool `json:"events"`
}

// ClaudeSettings represents the full .claude/settings.json structure.
// Only the hooks field is parsed; other fields are ignored.
type ClaudeSettings struct {
    Hooks *HookSettings `json:"hooks,omitempty"`
}

// AuthConfig represents the ~/.cops/auth.json structure.
type AuthConfig struct {
    Tokens     *TokenInfo `json:"tokens,omitempty"`
    HookAPIKey string     `json:"hookApiKey,omitempty"`
}

// TokenInfo contains OAuth token data.
type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    int64  `json:"expiresAt"`
}

// HookConfig represents the merged configuration for Hook functionality.
type HookConfig struct {
    Enabled     bool
    APIEndpoint string
    APIKey      string
    Events      map[EventType]bool
}

// IsEventEnabled returns true if the given event type is enabled.
func (c *HookConfig) IsEventEnabled(eventType EventType) bool {
    if !c.Enabled {
        return false
    }
    enabled, exists := c.Events[eventType]
    if !exists {
        // Default to enabled for unknown event types
        return true
    }
    return enabled
}

// Validate validates the HookConfig.
// Returns an error if the configuration is invalid.
func (c *HookConfig) Validate() error {
    if !c.Enabled {
        return nil // Disabled config is always valid
    }
    if c.APIEndpoint == "" {
        return ErrMissingAPIEndpoint
    }
    if c.APIKey == "" {
        return ErrMissingAPIKey
    }
    return nil
}
```

**Errors** (`shared/hookconfig/errors.go`):

```go
package hookconfig

import "errors"

var (
    ErrMissingAPIKey      = errors.New("hook API key is not configured in ~/.cops/auth.json")
    ErrMissingAPIEndpoint = errors.New("hook API endpoint is not configured in .claude/settings.json")
    ErrInvalidEventType   = errors.New("invalid event type")
)
```

### Task 2: Define Loader Port Interface

**File**: `shared/hookconfig/loader/loader_port.go`

```go
package loader

import (
    "context"

    "github.com/team-attention/cops/shared/hookconfig"
)

// HookConfigLoaderPort defines the interface for loading hook configuration.
type HookConfigLoaderPort interface {
    // Load loads and merges hook configuration from project and global files.
    // projectDir is the root directory of the project (containing .claude/).
    // Returns merged HookConfig or error if loading fails.
    Load(ctx context.Context, projectDir string) (*hookconfig.HookConfig, error)
}
```

### Task 3: Implement Filesystem Loader

**File**: `shared/hookconfig/loader/filesystem/filesystem_loader.go`

```go
package filesystem

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"

    "github.com/team-attention/cops/shared/hookconfig"
    "github.com/team-attention/cops/shared/hookconfig/loader"
)

// FilesystemLoader implements HookConfigLoaderPort using filesystem.
type FilesystemLoader struct {
    logger *slog.Logger
}

// NewFilesystemLoader creates a new filesystem-based hook config loader.
func NewFilesystemLoader(l *slog.Logger) loader.HookConfigLoaderPort {
    return &FilesystemLoader{
        logger: l.With(slog.String("name", "hookconfig.loader.filesystem")),
    }
}

// Load loads and merges hook configuration.
func (l *FilesystemLoader) Load(ctx context.Context, projectDir string) (*hookconfig.HookConfig, error) {
    // 1. Load project-level settings from .claude/settings.json
    claudeSettings, err := l.loadClaudeSettings(projectDir)
    if err != nil {
        return nil, fmt.Errorf("failed to load .claude/settings.json: %w", err)
    }

    // 2. Load global auth config from ~/.cops/auth.json
    authConfig, err := l.loadAuthConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to load ~/.cops/auth.json: %w", err)
    }

    // 3. Merge configurations
    config := l.mergeConfigs(claudeSettings, authConfig)

    // 4. Validate merged config
    if err := config.Validate(); err != nil {
        return nil, err
    }

    return config, nil
}

// loadClaudeSettings loads .claude/settings.json from the project directory.
func (l *FilesystemLoader) loadClaudeSettings(projectDir string) (*hookconfig.ClaudeSettings, error) {
    settingsPath := filepath.Join(projectDir, ".claude", "settings.json")

    // Check if file exists
    if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
        l.logger.Debug("no .claude/settings.json found, using defaults",
            slog.String("path", settingsPath),
        )
        return &hookconfig.ClaudeSettings{}, nil
    }

    // Read file
    data, err := os.ReadFile(settingsPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    // Parse JSON
    var settings hookconfig.ClaudeSettings
    if err := json.Unmarshal(data, &settings); err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %w", err)
    }

    return &settings, nil
}

// loadAuthConfig loads ~/.cops/auth.json.
func (l *FilesystemLoader) loadAuthConfig() (*hookconfig.AuthConfig, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("failed to get home directory: %w", err)
    }

    authPath := filepath.Join(homeDir, ".cops", "auth.json")

    // Check if file exists
    if _, err := os.Stat(authPath); os.IsNotExist(err) {
        l.logger.Debug("no ~/.cops/auth.json found",
            slog.String("path", authPath),
        )
        return &hookconfig.AuthConfig{}, nil
    }

    // Read file
    data, err := os.ReadFile(authPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    // Parse JSON
    var authConfig hookconfig.AuthConfig
    if err := json.Unmarshal(data, &authConfig); err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %w", err)
    }

    return &authConfig, nil
}

// mergeConfigs merges project and global configurations.
func (l *FilesystemLoader) mergeConfigs(
    claudeSettings *hookconfig.ClaudeSettings,
    authConfig *hookconfig.AuthConfig,
) *hookconfig.HookConfig {
    config := &hookconfig.HookConfig{
        Enabled: false,
        Events:  make(map[hookconfig.EventType]bool),
    }

    // Apply project-level hooks settings
    if claudeSettings.Hooks != nil {
        config.Enabled = claudeSettings.Hooks.Enabled
        config.APIEndpoint = claudeSettings.Hooks.APIEndpoint

        // Copy event settings
        for eventType, enabled := range claudeSettings.Hooks.Events {
            config.Events[eventType] = enabled
        }
    }

    // Apply global auth (API key)
    if authConfig != nil {
        config.APIKey = authConfig.HookAPIKey
    }

    // Set default events (all enabled) if none specified
    if len(config.Events) == 0 {
        for _, eventType := range hookconfig.AllEventTypes() {
            config.Events[eventType] = true
        }
    }

    return config
}

// Compile-time interface verification
var _ loader.HookConfigLoaderPort = (*FilesystemLoader)(nil)
```

### Task 4: Write Unit Tests

**File**: `shared/hookconfig/hookconfig_test.go`

```go
package hookconfig_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/team-attention/cops/shared/hookconfig"
)

func TestHookConfig_IsEventEnabled(t *testing.T) {
    tests := []struct {
        name      string
        config    *hookconfig.HookConfig
        eventType hookconfig.EventType
        expected  bool
    }{
        {
            name: "enabled event returns true",
            config: &hookconfig.HookConfig{
                Enabled: true,
                Events: map[hookconfig.EventType]bool{
                    hookconfig.EventTypePostToolUse: true,
                },
            },
            eventType: hookconfig.EventTypePostToolUse,
            expected:  true,
        },
        {
            name: "disabled event returns false",
            config: &hookconfig.HookConfig{
                Enabled: true,
                Events: map[hookconfig.EventType]bool{
                    hookconfig.EventTypePostToolUse: false,
                },
            },
            eventType: hookconfig.EventTypePostToolUse,
            expected:  false,
        },
        {
            name: "unknown event defaults to enabled",
            config: &hookconfig.HookConfig{
                Enabled: true,
                Events:  map[hookconfig.EventType]bool{},
            },
            eventType: hookconfig.EventTypePostToolUse,
            expected:  true,
        },
        {
            name: "disabled config returns false for all events",
            config: &hookconfig.HookConfig{
                Enabled: false,
                Events: map[hookconfig.EventType]bool{
                    hookconfig.EventTypePostToolUse: true,
                },
            },
            eventType: hookconfig.EventTypePostToolUse,
            expected:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.config.IsEventEnabled(tt.eventType)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestHookConfig_Validate(t *testing.T) {
    tests := []struct {
        name        string
        config      *hookconfig.HookConfig
        expectedErr error
    }{
        {
            name: "valid enabled config",
            config: &hookconfig.HookConfig{
                Enabled:     true,
                APIEndpoint: "https://api.example.com",
                APIKey:      "test-key",
            },
            expectedErr: nil,
        },
        {
            name: "disabled config is always valid",
            config: &hookconfig.HookConfig{
                Enabled: false,
            },
            expectedErr: nil,
        },
        {
            name: "missing API endpoint",
            config: &hookconfig.HookConfig{
                Enabled: true,
                APIKey:  "test-key",
            },
            expectedErr: hookconfig.ErrMissingAPIEndpoint,
        },
        {
            name: "missing API key",
            config: &hookconfig.HookConfig{
                Enabled:     true,
                APIEndpoint: "https://api.example.com",
            },
            expectedErr: hookconfig.ErrMissingAPIKey,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            assert.Equal(t, tt.expectedErr, err)
        })
    }
}
```

**File**: `shared/hookconfig/loader/filesystem/filesystem_loader_test.go`

```go
package filesystem_test

import (
    "context"
    "log/slog"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/team-attention/cops/shared/hookconfig"
    "github.com/team-attention/cops/shared/hookconfig/loader/filesystem"
)

func TestFilesystemLoader_Load(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    t.Run("loads complete configuration", func(t *testing.T) {
        // Setup temp directories
        projectDir := t.TempDir()
        homeDir := t.TempDir()

        // Create .claude/settings.json
        claudeDir := filepath.Join(projectDir, ".claude")
        require.NoError(t, os.MkdirAll(claudeDir, 0755))

        settingsJSON := `{
            "hooks": {
                "enabled": true,
                "apiEndpoint": "https://api.example.com",
                "events": {
                    "PostToolUse": true,
                    "Notification": false
                }
            }
        }`
        require.NoError(t, os.WriteFile(
            filepath.Join(claudeDir, "settings.json"),
            []byte(settingsJSON),
            0644,
        ))

        // Create ~/.cops/auth.json
        copsDir := filepath.Join(homeDir, ".cops")
        require.NoError(t, os.MkdirAll(copsDir, 0700))

        authJSON := `{"hookApiKey": "test-api-key"}`
        require.NoError(t, os.WriteFile(
            filepath.Join(copsDir, "auth.json"),
            []byte(authJSON),
            0600,
        ))

        // Override home directory for test
        originalHome := os.Getenv("HOME")
        os.Setenv("HOME", homeDir)
        defer os.Setenv("HOME", originalHome)

        // Load configuration
        loader := filesystem.NewFilesystemLoader(logger)
        config, err := loader.Load(context.Background(), projectDir)

        require.NoError(t, err)
        assert.True(t, config.Enabled)
        assert.Equal(t, "https://api.example.com", config.APIEndpoint)
        assert.Equal(t, "test-api-key", config.APIKey)
        assert.True(t, config.Events[hookconfig.EventTypePostToolUse])
        assert.False(t, config.Events[hookconfig.EventTypeNotification])
    })

    t.Run("returns error for missing API key when enabled", func(t *testing.T) {
        projectDir := t.TempDir()
        homeDir := t.TempDir()

        // Create .claude/settings.json with hooks enabled
        claudeDir := filepath.Join(projectDir, ".claude")
        require.NoError(t, os.MkdirAll(claudeDir, 0755))

        settingsJSON := `{
            "hooks": {
                "enabled": true,
                "apiEndpoint": "https://api.example.com"
            }
        }`
        require.NoError(t, os.WriteFile(
            filepath.Join(claudeDir, "settings.json"),
            []byte(settingsJSON),
            0644,
        ))

        // Create empty ~/.cops/auth.json (no hookApiKey)
        copsDir := filepath.Join(homeDir, ".cops")
        require.NoError(t, os.MkdirAll(copsDir, 0700))
        require.NoError(t, os.WriteFile(
            filepath.Join(copsDir, "auth.json"),
            []byte("{}"),
            0600,
        ))

        // Override home directory for test
        originalHome := os.Getenv("HOME")
        os.Setenv("HOME", homeDir)
        defer os.Setenv("HOME", originalHome)

        // Load configuration
        loader := filesystem.NewFilesystemLoader(logger)
        _, err := loader.Load(context.Background(), projectDir)

        assert.ErrorIs(t, err, hookconfig.ErrMissingAPIKey)
    })

    t.Run("disabled hooks are valid without API key", func(t *testing.T) {
        projectDir := t.TempDir()
        homeDir := t.TempDir()

        // Create .claude/settings.json with hooks disabled
        claudeDir := filepath.Join(projectDir, ".claude")
        require.NoError(t, os.MkdirAll(claudeDir, 0755))

        settingsJSON := `{
            "hooks": {
                "enabled": false
            }
        }`
        require.NoError(t, os.WriteFile(
            filepath.Join(claudeDir, "settings.json"),
            []byte(settingsJSON),
            0644,
        ))

        // Override home directory for test
        originalHome := os.Getenv("HOME")
        os.Setenv("HOME", homeDir)
        defer os.Setenv("HOME", originalHome)

        // Load configuration
        loader := filesystem.NewFilesystemLoader(logger)
        config, err := loader.Load(context.Background(), projectDir)

        require.NoError(t, err)
        assert.False(t, config.Enabled)
    })
}
```

### Task 5: Integration with Existing Auth State

The existing `cli/internal/platform/outbound/authstate/filesystem/authstate.go` already uses `~/.cops/auth.json` for OAuth tokens. We need to ensure:

1. The `AuthState` struct can be extended with `HookAPIKey` field
2. Backward compatibility with existing auth.json files (optional field)

**Update**: `cli/internal/platform/outbound/authstate/filesystem/authstate.go`

```go
// AuthState represents the local authentication state.
type AuthState struct {
    Tokens     *TokenInfo `json:"tokens,omitempty"`
    HookAPIKey string     `json:"hookApiKey,omitempty"` // NEW: Hook API key
}
```

## Implementation Order

1. **Task 1**: Define domain models (`shared/hookconfig/hookconfig.go`, `errors.go`)
2. **Task 2**: Define loader port interface (`shared/hookconfig/loader/loader_port.go`)
3. **Task 3**: Implement filesystem loader (`shared/hookconfig/loader/filesystem/filesystem_loader.go`)
4. **Task 4**: Write unit tests
5. **Task 5**: Update existing auth state structure (optional field)

## Testing Strategy

1. **Unit Tests**:
   - Test `HookConfig.IsEventEnabled()` with various scenarios
   - Test `HookConfig.Validate()` with valid/invalid configs
   - Test filesystem loader with mock file system

2. **Integration Tests**:
   - Test loading from real file system with temp directories
   - Test merging of project and global configs

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing auth.json | Use `omitempty` for new field, backward compatible |
| Race conditions on config load | Config loading is single-threaded per process |
| Invalid event type strings | Define constants and validation |

## Dependencies

- **TA-137** (Hook event protocol): Defines the event types to filter
- **TA-139** (API key authentication): Defines how API keys are issued/validated

## Estimated Effort

| Task | Estimate |
|------|----------|
| Task 1: Domain models | 1 hour |
| Task 2: Loader port | 0.5 hour |
| Task 3: Filesystem loader | 2 hours |
| Task 4: Unit tests | 2 hours |
| Task 5: Auth state update | 0.5 hour |
| **Total** | **6 hours** |
