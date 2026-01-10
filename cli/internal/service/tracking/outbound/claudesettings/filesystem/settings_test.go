package filesystem_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/claudesettings"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/claudesettings/filesystem"
)

// newTestLogger creates a logger for tests.
// Algorithm: Create slog with text handler to stdout
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// createSettingsFile is a test helper to create .claude/settings.json.
// Algorithm:
//  1. Create .claude directory
//  2. Write content to settings.json
func createSettingsFile(t *testing.T, projectDir string, content string) {
	t.Helper()
	claudeDir := filepath.Join(projectDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "settings.json"),
		[]byte(content),
		0644,
	))
}

// readSettingsFile is a test helper to read and return raw JSON.
// Algorithm: Read file and return bytes
func readSettingsFile(t *testing.T, projectDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	require.NoError(t, err)
	return data
}

func TestFilesystemClaudeSettings_Load(t *testing.T) {
	logger := newTestLogger()

	t.Run("returns empty settings when file not exists", func(t *testing.T) {
		// Arrange: Create empty temp directory
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Load from directory without .claude
		settings, err := adapter.Load(context.Background(), projectDir)

		// Assert: Empty settings, no error
		require.NoError(t, err)
		assert.NotNil(t, settings)
		assert.Nil(t, settings.Hooks)
	})

	t.Run("loads hooks from existing file", func(t *testing.T) {
		// Arrange: Create settings file with hooks
		projectDir := t.TempDir()
		content := `{
            "hooks": {
                "PostToolUse": [
                    {"matcher": "*", "hooks": [{"type": "command", "command": "echo test"}]}
                ]
            }
        }`
		createSettingsFile(t, projectDir, content)
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Load settings
		settings, err := adapter.Load(context.Background(), projectDir)

		// Assert: Hooks parsed correctly
		require.NoError(t, err)
		assert.Len(t, settings.Hooks["PostToolUse"], 1)
		assert.Equal(t, "*", settings.Hooks["PostToolUse"][0].Matcher)
		assert.Equal(t, "echo test", settings.Hooks["PostToolUse"][0].Hooks[0].Command)
	})

	t.Run("preserves other fields in rawFields", func(t *testing.T) {
		// Arrange: Create settings file with hooks and other fields
		projectDir := t.TempDir()
		content := `{
            "permissions": {"allow": ["Read"]},
            "hooks": {
                "SessionStart": []
            }
        }`
		createSettingsFile(t, projectDir, content)
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Load settings
		settings, err := adapter.Load(context.Background(), projectDir)

		// Assert: Other fields preserved
		require.NoError(t, err)
		rawFields := settings.GetRawFields()
		assert.NotNil(t, rawFields["permissions"])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		// Arrange: Create invalid JSON file
		projectDir := t.TempDir()
		createSettingsFile(t, projectDir, "not valid json")
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Load settings
		settings, err := adapter.Load(context.Background(), projectDir)

		// Assert: Error returned
		assert.Error(t, err)
		assert.Nil(t, settings)
	})
}

func TestFilesystemClaudeSettings_Save(t *testing.T) {
	logger := newTestLogger()

	t.Run("creates .claude directory if not exists", func(t *testing.T) {
		// Arrange: Empty project directory
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)
		settings := &claudesettings.ClaudeSettings{
			Hooks: map[string][]*claudesettings.HookEntry{
				"SessionStart": {},
			},
		}

		// Act: Save settings
		err := adapter.Save(context.Background(), projectDir, settings)

		// Assert: Directory and file created
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(projectDir, ".claude", "settings.json"))
		assert.NoError(t, err)
	})

	t.Run("preserves rawFields in output", func(t *testing.T) {
		// Arrange: Settings with rawFields
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)
		settings := &claudesettings.ClaudeSettings{
			Hooks: map[string][]*claudesettings.HookEntry{},
		}
		settings.SetRawFields(map[string]interface{}{
			"permissions": map[string]interface{}{"allow": []string{"Read"}},
		})

		// Act: Save and re-read
		require.NoError(t, adapter.Save(context.Background(), projectDir, settings))
		data := readSettingsFile(t, projectDir)

		// Assert: Other fields present in output
		var output map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &output))
		assert.NotNil(t, output["permissions"])
	})
}

func TestFilesystemClaudeSettings_InstallCopsHooks(t *testing.T) {
	logger := newTestLogger()

	t.Run("installs all 7 event types on empty project", func(t *testing.T) {
		// Arrange: Empty project directory
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Install hooks
		err := adapter.InstallCopsHooks(context.Background(), projectDir)

		// Assert: All event types have cops hooks
		require.NoError(t, err)
		settings, err := adapter.Load(context.Background(), projectDir)
		require.NoError(t, err)

		for _, eventType := range claudesettings.AllEventTypes {
			require.Len(t, settings.Hooks[eventType], 1,
				"event type %s should have 1 hook entry", eventType)
			assert.Contains(t, settings.Hooks[eventType][0].Hooks[0].Command, "cops hook post")
		}
	})

	t.Run("uses * matcher for PostToolUse", func(t *testing.T) {
		// Arrange
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))

		// Assert
		settings, _ := adapter.Load(context.Background(), projectDir)
		assert.Equal(t, "*", settings.Hooks["PostToolUse"][0].Matcher)
	})

	t.Run("uses empty matcher for other events", func(t *testing.T) {
		// Arrange
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))

		// Assert: Check non-PostToolUse events
		settings, _ := adapter.Load(context.Background(), projectDir)
		for _, eventType := range []string{"SessionStart", "SessionEnd", "Stop"} {
			assert.Equal(t, "", settings.Hooks[eventType][0].Matcher,
				"event type %s should have empty matcher", eventType)
		}
	})

	t.Run("preserves existing hooks", func(t *testing.T) {
		// Arrange: Project with existing hooks
		projectDir := t.TempDir()
		content := `{
            "hooks": {
                "PostToolUse": [
                    {"matcher": "*", "hooks": [{"type": "command", "command": "echo existing"}]}
                ]
            }
        }`
		createSettingsFile(t, projectDir, content)
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Install cops hooks
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))

		// Assert: Both existing and cops hooks present
		settings, _ := adapter.Load(context.Background(), projectDir)
		assert.Len(t, settings.Hooks["PostToolUse"], 2)
		assert.Equal(t, "echo existing", settings.Hooks["PostToolUse"][0].Hooks[0].Command)
		assert.Contains(t, settings.Hooks["PostToolUse"][1].Hooks[0].Command, "cops hook post")
	})

	t.Run("does not duplicate cops hooks on repeated install", func(t *testing.T) {
		// Arrange
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act: Install twice
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))

		// Assert: Still only one cops hook per event
		settings, _ := adapter.Load(context.Background(), projectDir)
		for _, eventType := range claudesettings.AllEventTypes {
			assert.Len(t, settings.Hooks[eventType], 1,
				"event type %s should have exactly 1 hook entry after double install", eventType)
		}
	})

	t.Run("preserves other settings fields", func(t *testing.T) {
		// Arrange: Settings with permissions
		projectDir := t.TempDir()
		content := `{
            "permissions": {"allow": ["Read", "Write"]},
            "enableAllProjectMcpServers": true
        }`
		createSettingsFile(t, projectDir, content)
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))

		// Assert: Other fields preserved
		data := readSettingsFile(t, projectDir)
		var output map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &output))
		assert.NotNil(t, output["permissions"])
		assert.NotNil(t, output["enableAllProjectMcpServers"])
		assert.NotNil(t, output["hooks"])
	})
}

func TestFilesystemClaudeSettings_HasCopsHooks(t *testing.T) {
	logger := newTestLogger()

	t.Run("returns false when no hooks installed", func(t *testing.T) {
		// Arrange
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act
		hasHooks, err := adapter.HasCopsHooks(context.Background(), projectDir)

		// Assert
		require.NoError(t, err)
		assert.False(t, hasHooks)
	})

	t.Run("returns true when cops hooks installed", func(t *testing.T) {
		// Arrange
		projectDir := t.TempDir()
		adapter := filesystem.NewFilesystemClaudeSettings(logger)
		require.NoError(t, adapter.InstallCopsHooks(context.Background(), projectDir))

		// Act
		hasHooks, err := adapter.HasCopsHooks(context.Background(), projectDir)

		// Assert
		require.NoError(t, err)
		assert.True(t, hasHooks)
	})

	t.Run("returns false when only other hooks present", func(t *testing.T) {
		// Arrange
		projectDir := t.TempDir()
		content := `{
            "hooks": {
                "PostToolUse": [
                    {"matcher": "*", "hooks": [{"type": "command", "command": "echo other"}]}
                ]
            }
        }`
		createSettingsFile(t, projectDir, content)
		adapter := filesystem.NewFilesystemClaudeSettings(logger)

		// Act
		hasHooks, err := adapter.HasCopsHooks(context.Background(), projectDir)

		// Assert
		require.NoError(t, err)
		assert.False(t, hasHooks)
	})
}
