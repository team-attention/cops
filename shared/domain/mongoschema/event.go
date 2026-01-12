package mongoschema

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	EventCollectionName = "events"
)

// Event-level fields
const (
	EventIDField            = "_id"
	EventHookEventNameField = "hookEventName"
	EventUserIDField        = "userId"
	EventSessionIDField     = "sessionId"
	EventCreatedAtField     = "createdAt"
	EventProjectIDField     = "projectId"
)

// Common fields (HookEventBase)
const (
	HookEventSessionIDField      = "sessionId"
	HookEventPermissionModeField = "permissionMode"
	// HookEventTranscriptPathField = "transcriptPath" // Excluded: local file path, not needed on server
	// HookEventCwdField            = "cwd"            // Excluded: local file path, not needed on server
)

// PreToolUse-specific fields
const (
	PreToolUseToolNameField  = "toolName"
	PreToolUseToolInputField = "toolInput"
	PreToolUseToolUseIDField = "toolUseId"
)

// PostToolUse-specific fields
const (
	PostToolUseToolNameField     = "toolName"
	PostToolUseToolInputField    = "toolInput"
	PostToolUseToolResponseField = "toolResponse"
	PostToolUseToolUseIDField    = "toolUseId"
)

// Notification-specific fields
const (
	NotificationMessageField          = "message"
	NotificationNotificationTypeField = "notificationType"
)

// UserPromptSubmit-specific fields
const (
	UserPromptSubmitPromptField = "prompt"
)

// Stop-specific fields
const (
	StopStopHookActiveField = "stopHookActive"
)

// SubagentStop-specific fields
const (
	SubagentStopStopHookActiveField = "stopHookActive"
	SubagentStopAgentIDField        = "agentId"
	// SubagentStopAgentTranscriptPathField = "agentTranscriptPath" // Excluded: local file path, not needed on server
)

// PreCompact-specific fields
const (
	PreCompactTriggerField            = "trigger"
	PreCompactCustomInstructionsField = "customInstructions"
)

// SessionStart-specific fields
const (
	SessionStartSourceField = "source"
)

// SessionEnd-specific fields
const (
	SessionEndReasonField = "reason"
)

// snakeToCamelSpecialCases maps JSON field names that don't follow standard snake_case → camelCase conversion.
// Currently empty as all fields follow the standard pattern.
var snakeToCamelSpecialCases = map[string]string{}

// snakeToCamel converts snake_case string to camelCase.
// e.g., "session_id" -> "sessionId", "tool_name" -> "toolName"
func snakeToCamel(s string) string {
	// Check for special case mappings first
	if mapped, ok := snakeToCamelSpecialCases[s]; ok {
		return mapped
	}

	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// convertKeysToCamel recursively converts all map keys from snake_case to camelCase.
func convertKeysToCamel(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			newKey := snakeToCamel(k)
			result[newKey] = convertKeysToCamel(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertKeysToCamel(item)
		}
		return result
	default:
		return v
	}
}

// Event is the MongoDB schema wrapper for domain.Event.
type Event struct {
	domain.Event `bson:"-"` // Handled via ToBSONDocument() to avoid BSON inline limitation with any type
	ID           bson.ObjectID `bson:"_id,omitempty"`
	UserID       bson.ObjectID `bson:"userId"`
	SessionID    string        `bson:"sessionId"`
	CreatedAt    time.Time     `bson:"createdAt"`
}

// ToBSONDocument converts Event to bson.M for MongoDB insertion.
// Uses JSON as intermediate format to handle polymorphic Data field,
// then converts snake_case keys to camelCase for MongoDB storage.
func (s *Event) ToBSONDocument() (bson.M, error) {
	// Marshal domain.Event to JSON (uses custom MarshalJSON)
	jsonBytes, err := json.Marshal(s.Event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	// Unmarshal JSON into map[string]any
	var rawDoc map[string]any
	if err := json.Unmarshal(jsonBytes, &rawDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert snake_case keys to camelCase for MongoDB storage
	camelDoc := convertKeysToCamel(rawDoc).(map[string]any)

	// Convert to bson.M
	doc := bson.M(camelDoc)

	// Add mongoschema-specific fields
	if !s.ID.IsZero() {
		doc["_id"] = s.ID
	}
	doc["userId"] = s.UserID
	doc["sessionId"] = s.SessionID
	doc["createdAt"] = s.CreatedAt

	return doc, nil
}

// FromDomain converts a domain.Event to MongoDB schema.
func (s *Event) FromDomain(userID string, d *domain.Event) {
	if d == nil {
		return
	}

	s.Event = *d

	if userID != "" {
		s.UserID, _ = bson.ObjectIDFromHex(userID)
	}

	// Extract SessionID from HookEventBase and store as string
	if base := s.getHookEventBase(); base != nil {
		s.SessionID = base.SessionID
	}

	s.CreatedAt = time.Now()
}

// ToDomain converts MongoDB schema to domain.Event.
func (s *Event) ToDomain() *domain.Event {
	if s == nil {
		return nil
	}

	// Set SessionID back into HookEventBase
	if base := s.getHookEventBase(); base != nil {
		base.SessionID = s.SessionID
	}

	return &s.Event
}

// getHookEventBase returns a pointer to the HookEventBase from the Event.Data field.
// Uses type switch to handle all known event types.
func (s *Event) getHookEventBase() *domain.HookEventBase {
	if s.Data == nil {
		return nil
	}

	switch data := s.Data.(type) {
	case *domain.PreToolUseEvent:
		return &data.HookEventBase
	case *domain.PostToolUseEvent:
		return &data.HookEventBase
	case *domain.NotificationEvent:
		return &data.HookEventBase
	case *domain.UserPromptSubmitEvent:
		return &data.HookEventBase
	case *domain.StopEvent:
		return &data.HookEventBase
	case *domain.SubagentStopEvent:
		return &data.HookEventBase
	case *domain.PreCompactEvent:
		return &data.HookEventBase
	case *domain.SessionStartEvent:
		return &data.HookEventBase
	case *domain.SessionEndEvent:
		return &data.HookEventBase
	default:
		return nil
	}
}
