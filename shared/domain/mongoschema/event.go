package mongoschema

import (
	"time"

	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	EventCollectionName = "events"
)

// Event-level fields
const (
	EventIDField        = "_id"
	EventTypeField      = "type"
	EventUserIDField    = "userId"
	EventSessionIDField = "sessionId"
	EventTimestampField = "timestamp"
	EventCreatedAtField = "createdAt"
)

// SessionStart-specific fields
const (
	SessionStartSessionTypeField = "sessionType"
	SessionStartToolsField       = "tools"
	SessionStartMcpServersField  = "mcpServers"
	SessionStartModelField       = "model"
	SessionStartPermModeField    = "permMode"
	SessionStartMaxTurnsField    = "maxTurns"
)

// PostToolUse-specific fields
const (
	PostToolUseToolNameField   = "toolName"
	PostToolUseToolIDField     = "toolId"
	PostToolUseSuccessField    = "success"
	PostToolUseErrorField      = "error"
	PostToolUseDurationMsField = "durationMs"
)

// Notification-specific fields
const (
	NotificationLevelField    = "level"
	NotificationMessageField  = "message"
	NotificationCategoryField = "category"
)

// Stop-specific fields
const (
	StopStopReasonField   = "stopReason"
	StopTotalTurnsField   = "totalTurns"
	StopInputTokensField  = "inputTokens"
	StopOutputTokensField = "outputTokens"
)

// SessionEnd-specific fields
const (
	SessionEndExitCodeField          = "exitCode"
	SessionEndTotalDurationMsField   = "totalDurationMs"
	SessionEndTotalInputTokensField  = "totalInputTokens"
	SessionEndTotalOutputTokensField = "totalOutputTokens"
)

// Event is the MongoDB schema wrapper for domain.Event.
type Event struct {
	domain.Event `bson:",inline"`
	ID           bson.ObjectID `bson:"_id,omitempty"`
	UserID       bson.ObjectID `bson:"userId"`
	SessionID    string        `bson:"sessionId"`
	CreatedAt    time.Time     `bson:"createdAt"`
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

	// Extract SessionID from EventBase and store as string
	if base := s.getEventBase(); base != nil {
		s.SessionID = string(base.SessionID)
	}

	s.CreatedAt = time.Now()
}

// ToDomain converts MongoDB schema to domain.Event.
func (s *Event) ToDomain() *domain.Event {
	if s == nil {
		return nil
	}

	// Set SessionID back into EventBase
	if base := s.getEventBase(); base != nil {
		base.SessionID = domain.ID(s.SessionID)
	}

	return &s.Event
}

// getEventBase returns a pointer to the EventBase from the Event.Data field.
// Uses type switch to handle all known event types.
func (s *Event) getEventBase() *domain.EventBase {
	if s.Data == nil {
		return nil
	}

	switch data := s.Data.(type) {
	case *domain.SessionStartEvent:
		return &data.EventBase
	case *domain.PostToolUseEvent:
		return &data.EventBase
	case *domain.NotificationEvent:
		return &data.EventBase
	case *domain.UserPromptSubmitEvent:
		return &data.EventBase
	case *domain.StopEvent:
		return &data.EventBase
	case *domain.SubagentStopEvent:
		return &data.EventBase
	case *domain.SessionEndEvent:
		return &data.EventBase
	default:
		return nil
	}
}
