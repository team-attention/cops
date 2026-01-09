package hookconfig

import "context"

// EventType represents the type of Hook event.
type EventType string

const (
	EventTypePostToolUse      EventType = "PostToolUse"
	EventTypeNotification     EventType = "Notification"
	EventTypeUserPromptSubmit EventType = "UserPromptSubmit"
	EventTypeStop             EventType = "Stop"
	EventTypeSubagentStop     EventType = "SubagentStop"
	EventTypeSessionStart     EventType = "SessionStart"
	EventTypeSessionEnd       EventType = "SessionEnd"
)

// AllEventTypes returns all supported event types.
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

// HookSettings represents the Hook configuration in .claude/settings.json.
// This is embedded within the Claude settings file under "cops" key.
type HookSettings struct {
	// Enabled indicates whether Hook event sending is enabled for this project.
	Enabled bool `json:"enabled"`

	// Events specifies which event types to send.
	// If nil or empty, all events are enabled when Enabled is true.
	Events *EventConfig `json:"events,omitempty"`
}

// EventConfig specifies individual event type settings.
// All fields are pointers to distinguish between "not set" (nil) and "explicitly false".
type EventConfig struct {
	PostToolUse      *bool `json:"postToolUse,omitempty"`
	Notification     *bool `json:"notification,omitempty"`
	UserPromptSubmit *bool `json:"userPromptSubmit,omitempty"`
	Stop             *bool `json:"stop,omitempty"`
	SubagentStop     *bool `json:"subagentStop,omitempty"`
	SessionStart     *bool `json:"sessionStart,omitempty"`
	SessionEnd       *bool `json:"sessionEnd,omitempty"`
}

// ClaudeSettings represents the structure of .claude/settings.json.
// We only parse the "cops" key, leaving other settings untouched.
type ClaudeSettings struct {
	Cops *HookSettings `json:"cops,omitempty"`
}

// AuthConfig represents the API key configuration in ~/.cops/auth.json.
type AuthConfig struct {
	// APIKey is the API key for authenticating with the C-Ops API server.
	APIKey string `json:"apiKey"`

	// APIEndpoint is the optional custom API endpoint URL.
	// If nil, uses the default endpoint from environment or config.
	APIEndpoint *string `json:"apiEndpoint,omitempty"`
}

// Config is the merged configuration from both files.
type Config struct {
	// Hook contains the project-specific Hook settings.
	Hook *HookSettings

	// Auth contains the API authentication configuration.
	Auth *AuthConfig
}

// IsEventEnabled checks if a specific event type is enabled.
// Returns true if:
// - Hook is enabled AND
// - Events config is nil (all events enabled) OR
// - The specific event type is not explicitly set to false
func (c *Config) IsEventEnabled(eventType EventType) bool {
	if c.Hook == nil || !c.Hook.Enabled {
		return false
	}

	if c.Hook.Events == nil {
		return true // All events enabled by default
	}

	return c.isEventTypeEnabled(eventType)
}

// isEventTypeEnabled checks the specific event type configuration.
func (c *Config) isEventTypeEnabled(eventType EventType) bool {
	events := c.Hook.Events

	switch eventType {
	case EventTypePostToolUse:
		return events.PostToolUse == nil || *events.PostToolUse
	case EventTypeNotification:
		return events.Notification == nil || *events.Notification
	case EventTypeUserPromptSubmit:
		return events.UserPromptSubmit == nil || *events.UserPromptSubmit
	case EventTypeStop:
		return events.Stop == nil || *events.Stop
	case EventTypeSubagentStop:
		return events.SubagentStop == nil || *events.SubagentStop
	case EventTypeSessionStart:
		return events.SessionStart == nil || *events.SessionStart
	case EventTypeSessionEnd:
		return events.SessionEnd == nil || *events.SessionEnd
	default:
		return true
	}
}

// IsEnabled returns true if Hook feature is enabled.
func (c *Config) IsEnabled() bool {
	return c.Hook != nil && c.Hook.Enabled
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Hook != nil && c.Hook.Enabled {
		if c.Auth == nil || c.Auth.APIKey == "" {
			return ErrAPIKeyRequired
		}
	}
	return nil
}

// HookConfigPort defines the interface for loading Hook configuration.
type HookConfigPort interface {
	// LoadConfig loads and merges Hook configuration from project and global sources.
	// projectDir is the path to the project directory containing .claude/settings.json.
	// Returns merged Config or error if loading fails.
	LoadConfig(ctx context.Context, projectDir string) (*Config, error)

	// LoadHookSettings loads only the Hook settings from .claude/settings.json.
	// Returns nil if file doesn't exist or "cops" key is not present.
	LoadHookSettings(ctx context.Context, projectDir string) (*HookSettings, error)

	// LoadAuthConfig loads the API key configuration from ~/.cops/auth.json.
	// Returns nil if file doesn't exist.
	LoadAuthConfig(ctx context.Context) (*AuthConfig, error)
}
