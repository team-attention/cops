package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/claudesettings"
)

const (
	claudeDirName      = ".claude"
	settingsFileName   = "settings.json"
	copsHookCommand    = "cops hook post"
	postToolUseMatcher = "*" // PostToolUse uses * matcher
	defaultMatcher     = ""  // Other events use empty matcher
)

// FilesystemClaudeSettings implements ClaudeSettingsPort using filesystem.
type FilesystemClaudeSettings struct {
	logger *slog.Logger
}

// NewFilesystemClaudeSettings creates a new filesystem-based Claude settings adapter.
// Algorithm:
//  1. Create struct with logger bound to "tracking.claudesettings.filesystem"
//  2. Return pointer to struct
func NewFilesystemClaudeSettings(l *slog.Logger) *FilesystemClaudeSettings {
	return &FilesystemClaudeSettings{
		logger: l.With(slog.String("name", "tracking.claudesettings.filesystem")),
	}
}

// Load reads existing settings from .claude/settings.json.
// Algorithm:
//  1. Build settingsPath = projectPath + "/.claude/settings.json"
//  2. Check if file exists with os.Stat
//  3. If not exists (os.IsNotExist), return empty ClaudeSettings{Hooks: nil}
//  4. Read file content with os.ReadFile
//  5. Unmarshal into map[string]json.RawMessage to preserve unknown fields
//  6. Extract "hooks" raw message and unmarshal into map[string][]*HookEntry
//  7. Delete "hooks" from raw map
//  8. Convert remaining raw map entries to interface{} map for rawFields
//  9. Create ClaudeSettings, set Hooks and rawFields
//  10. Return settings
func (f *FilesystemClaudeSettings) Load(ctx context.Context, projectPath string) (*claudesettings.ClaudeSettings, error) {
	settingsPath := filepath.Join(projectPath, claudeDirName, settingsFileName)

	// Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		f.logger.Debug("settings.json not found, returning empty settings",
			slog.String("path", settingsPath),
		)
		return &claudesettings.ClaudeSettings{}, nil
	}

	// Read file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings.json: %w", err)
	}

	// Parse into intermediate map preserving raw messages
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("failed to parse settings.json: %w", err)
	}

	settings := &claudesettings.ClaudeSettings{}

	// Extract hooks if present
	if hooksRaw, ok := rawMap["hooks"]; ok {
		var hooks map[string][]*claudesettings.HookEntry
		if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
			return nil, fmt.Errorf("failed to parse hooks: %w", err)
		}
		settings.Hooks = hooks
		delete(rawMap, "hooks")
	}

	// Convert remaining fields to interface{} map
	if len(rawMap) > 0 {
		rawFields := make(map[string]interface{})
		for k, v := range rawMap {
			var val interface{}
			if err := json.Unmarshal(v, &val); err != nil {
				return nil, fmt.Errorf("failed to parse field %s: %w", k, err)
			}
			rawFields[k] = val
		}
		settings.SetRawFields(rawFields)
	}

	f.logger.Debug("loaded settings.json",
		slog.String("path", settingsPath),
		slog.Int("hookEventTypes", len(settings.Hooks)),
	)

	return settings, nil
}

// Save writes settings to .claude/settings.json.
// Algorithm:
//  1. Build claudeDir = projectPath + "/.claude"
//  2. Create directory with os.MkdirAll(claudeDir, 0755)
//  3. Build output map starting with rawFields
//  4. If Hooks not empty, add "hooks" key
//  5. Marshal with json.MarshalIndent (prefix="", indent="  ")
//  6. Write with os.WriteFile (path, data, 0644)
func (f *FilesystemClaudeSettings) Save(ctx context.Context, projectPath string, settings *claudesettings.ClaudeSettings) error {
	claudeDir := filepath.Join(projectPath, claudeDirName)

	// Ensure .claude directory exists
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, settingsFileName)

	// Build output map preserving other fields
	output := make(map[string]interface{})
	if rawFields := settings.GetRawFields(); rawFields != nil {
		for k, v := range rawFields {
			output[k] = v
		}
	}

	// Add hooks if present
	if len(settings.Hooks) > 0 {
		output["hooks"] = settings.Hooks
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write file
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	f.logger.Info("saved settings.json",
		slog.String("path", settingsPath),
	)

	return nil
}

// InstallCopsHooks adds cops hook entries to all 7 event types.
// Algorithm:
//  1. Load existing settings
//  2. Initialize Hooks map if nil
//  3. For each event type:
//     a. Determine matcher ("*" for PostToolUse, "" for others)
//     b. Check if cops hook already exists for this event
//     c. If not exists, create new HookEntry and append
//  4. Save settings
func (f *FilesystemClaudeSettings) InstallCopsHooks(ctx context.Context, projectPath string) error {
	// Load existing settings
	settings, err := f.Load(ctx, projectPath)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Initialize hooks map if nil
	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]*claudesettings.HookEntry)
	}

	// Add cops hooks for each event type
	for _, eventType := range claudesettings.AllEventTypes {
		// Skip if cops hook already exists
		if f.hasCopsHookForEvent(settings.Hooks[eventType]) {
			f.logger.Debug("cops hook already installed",
				slog.String("eventType", eventType),
			)
			continue
		}

		// Determine matcher based on event type
		matcher := defaultMatcher
		if eventType == "PostToolUse" {
			matcher = postToolUseMatcher
		}

		// Create new hook entry
		entry := &claudesettings.HookEntry{
			Matcher: matcher,
			Hooks: []*claudesettings.HookCommand{
				{
					Type:    "command",
					Command: copsHookCommand,
				},
			},
		}

		// Append to existing hooks (preserve other hooks)
		settings.Hooks[eventType] = append(settings.Hooks[eventType], entry)

		f.logger.Debug("added cops hook",
			slog.String("eventType", eventType),
			slog.String("matcher", matcher),
		)
	}

	// Save updated settings
	if err := f.Save(ctx, projectPath, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	f.logger.Info("installed cops hooks",
		slog.String("projectPath", projectPath),
	)

	return nil
}

// HasCopsHooks checks if any cops hooks are already installed.
// Algorithm:
//  1. Load settings
//  2. For each event type, check if any hook contains copsHookCommand
//  3. Return true on first match
func (f *FilesystemClaudeSettings) HasCopsHooks(ctx context.Context, projectPath string) (bool, error) {
	settings, err := f.Load(ctx, projectPath)
	if err != nil {
		return false, err
	}

	for _, eventType := range claudesettings.AllEventTypes {
		if f.hasCopsHookForEvent(settings.Hooks[eventType]) {
			return true, nil
		}
	}

	return false, nil
}

// hasCopsHookForEvent checks if any entry contains cops hook command.
// Algorithm:
//  1. Iterate over entries
//  2. For each entry, iterate over Hooks
//  3. Check if Command contains copsHookCommand string
//  4. Return true on first match
func (f *FilesystemClaudeSettings) hasCopsHookForEvent(entries []*claudesettings.HookEntry) bool {
	for _, entry := range entries {
		for _, hook := range entry.Hooks {
			if strings.Contains(hook.Command, copsHookCommand) {
				return true
			}
		}
	}
	return false
}

// Compile-time interface verification
var _ claudesettings.ClaudeSettingsPort = (*FilesystemClaudeSettings)(nil)
