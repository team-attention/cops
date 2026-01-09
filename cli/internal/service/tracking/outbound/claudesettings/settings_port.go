package claudesettings

import "context"

// HookEntry represents a single hook matcher entry in settings.json.
// Example: { "matcher": "", "hooks": [{"type": "command", "command": "cops hook post"}] }
type HookEntry struct {
	// Matcher is the pattern to match (empty string or "*" for all).
	// Algorithm: Store as-is, "" matches all for most events, "*" for PostToolUse
	Matcher string `json:"matcher"`

	// Hooks is the list of hook commands to execute.
	// Algorithm: Each entry has Type="command" and Command with the actual command
	Hooks []*HookCommand `json:"hooks"`
}

// HookCommand represents a single hook command.
// Example: {"type": "command", "command": "cops hook post"}
type HookCommand struct {
	// Type is the hook type, always "command" for CLI hooks.
	Type string `json:"type"`

	// Command is the shell command to execute.
	Command string `json:"command"`
}

// ClaudeSettings represents the full .claude/settings.json structure.
// Algorithm: Only hooks field is managed; other fields preserved via rawFields
type ClaudeSettings struct {
	// Hooks maps event type names (PascalCase) to their hook entries.
	// Event types: PostToolUse, Notification, UserPromptSubmit, Stop, SubagentStop, SessionStart, SessionEnd
	Hooks map[string][]*HookEntry `json:"hooks,omitempty"`

	// rawFields stores other JSON fields for preservation during save.
	// Algorithm: Populated during Load, merged back during Save
	rawFields map[string]interface{}
}

// SetRawFields stores non-hooks fields for later preservation.
// Algorithm: Called by adapter after parsing, stores map reference
func (s *ClaudeSettings) SetRawFields(fields map[string]interface{}) {
	s.rawFields = fields
}

// GetRawFields returns stored non-hooks fields.
// Algorithm: Called by adapter before saving, returns stored map
func (s *ClaudeSettings) GetRawFields() map[string]interface{} {
	return s.rawFields
}

// AllEventTypes returns all 7 Claude Code hook event type names in PascalCase.
// Algorithm: Return static slice of all supported event types
var AllEventTypes = []string{
	"PostToolUse",
	"Notification",
	"UserPromptSubmit",
	"Stop",
	"SubagentStop",
	"SessionStart",
	"SessionEnd",
}

// ClaudeSettingsPort defines the interface for managing .claude/settings.json.
type ClaudeSettingsPort interface {
	// Load reads existing settings from .claude/settings.json.
	// Algorithm:
	//   1. Build path: projectPath + "/.claude/settings.json"
	//   2. If file not exists, return empty ClaudeSettings with nil Hooks
	//   3. Read and parse JSON into intermediate map to preserve unknown fields
	//   4. Extract "hooks" field into ClaudeSettings.Hooks
	//   5. Store remaining fields in rawFields for preservation
	//   6. Return ClaudeSettings
	// Returns: nil error if file not exists (empty settings created)
	Load(ctx context.Context, projectPath string) (*ClaudeSettings, error)

	// Save writes settings to .claude/settings.json.
	// Algorithm:
	//   1. Build path: projectPath + "/.claude/settings.json"
	//   2. Ensure .claude directory exists (os.MkdirAll with 0755)
	//   3. Start with rawFields map (preserves other settings)
	//   4. Set "hooks" key with ClaudeSettings.Hooks
	//   5. Marshal with indent (2 spaces) and write with 0644 permissions
	// Returns: error if directory creation or write fails
	Save(ctx context.Context, projectPath string, settings *ClaudeSettings) error

	// InstallCopsHooks adds cops hook entries to all 7 event types.
	// Algorithm:
	//   1. Call Load(projectPath) to get existing settings
	//   2. Initialize Hooks map if nil
	//   3. For each event type in AllEventTypes:
	//      a. Check if cops hook already exists (contains "cops hook post")
	//      b. If not exists, create HookEntry with matcher and cops command
	//      c. Append to existing hooks for that event type
	//   4. Call Save(projectPath, settings)
	// Returns: error if load or save fails
	InstallCopsHooks(ctx context.Context, projectPath string) error

	// HasCopsHooks checks if any cops hooks are already installed.
	// Algorithm:
	//   1. Call Load(projectPath) to get existing settings
	//   2. For each event type, check if any hook command contains "cops hook post"
	//   3. Return true if at least one found
	// Returns: (bool, error) - true if cops hooks exist
	HasCopsHooks(ctx context.Context, projectPath string) (bool, error)
}
