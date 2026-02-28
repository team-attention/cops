package mongoschema

import (
	domain "github.com/team-attention/cops/shared/domain/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SessionsCollectionName is the MongoDB collection name for session documents.
const SessionsCollectionName = "sessions"

// Session collection field names.
// Naming pattern: Session<FieldName>Field
const (
	SessionIDField                 = "_id"
	SessionProjectIDField          = "projectId"
	SessionUserIDField             = "userId"
	SessionSessionIDField          = "sessionId"
	SessionAgentIDField            = "agentId"
	SessionSpawnedByToolUseIDField = "spawnedByToolUseId"
	SessionTypeField               = "type"
	SessionTimestampField          = "timestamp"
	SessionProviderField           = "provider"
	SessionVersionField            = "version"
	SessionGitBranchField          = "gitBranch"
	SessionIsSidechainField        = "isSidechain"
	SessionUUIDField               = "uuid"
)

// ToolExecution field names.
const (
	SessionToolNameField = "toolName"
)

// AgentMessage token usage field paths (for sessions collection).
// These are used for aggregation queries on agent message types.
const (
	SessionUsageField                        = "usage"
	SessionUsageInputTokensField             = "inputTokens"
	SessionUsageOutputTokensField            = "outputTokens"
	SessionUsageCacheCreationInputTokensField = "cacheCreationInputTokens"
	SessionUsageCacheReadInputTokensField    = "cacheReadInputTokens"
)

// Composite field paths for MongoDB queries on sessions collection.
const (
	SessionUsageInputTokensPath              = SessionUsageField + "." + SessionUsageInputTokensField
	SessionUsageOutputTokensPath             = SessionUsageField + "." + SessionUsageOutputTokensField
	SessionUsageCacheCreationInputTokensPath = SessionUsageField + "." + SessionUsageCacheCreationInputTokensField
	SessionUsageCacheReadInputTokensPath     = SessionUsageField + "." + SessionUsageCacheReadInputTokensField
)

// Session wraps domain.Session for MongoDB serialization.
// Handles the "Data any" field that cannot use bson:",inline" directly.
type Session struct {
	*domain.Session
	ProjectID bson.ObjectID `bson:"projectId"`
	UserID    bson.ObjectID `bson:"userId"`
}

// MarshalBSON flattens Data fields alongside "type" field for MongoDB storage.
func (s Session) MarshalBSON() ([]byte, error) {
	// Marshal Data to get its fields
	dataBytes, err := bson.Marshal(s.Session.Data)
	if err != nil {
		return nil, err
	}

	var doc bson.M
	if err := bson.Unmarshal(dataBytes, &doc); err != nil {
		return nil, err
	}

	// Add type and IDs
	doc[SessionTypeField] = s.Session.Type
	doc[SessionProjectIDField] = s.ProjectID
	doc[SessionUserIDField] = s.UserID

	return bson.Marshal(doc)
}

// UnmarshalBSON implements bson.Unmarshaler for polymorphic deserialization.
func (s *Session) UnmarshalBSON(data []byte) error {
	// 1. Extract type, projectId, and userId first
	var base struct {
		Type      domain.SessionType `bson:"type"`
		ProjectID bson.ObjectID      `bson:"projectId"`
		UserID    bson.ObjectID      `bson:"userId"`
	}
	if err := bson.Unmarshal(data, &base); err != nil {
		return err
	}
	s.Session = &domain.Session{Type: base.Type}
	s.ProjectID = base.ProjectID
	s.UserID = base.UserID

	// 2. Unmarshal to appropriate type based on discriminator
	switch base.Type {
	case domain.SessionTypeProgress:
		var v domain.Progress
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		s.Session.Data = &v
	case domain.SessionTypeHuman:
		var v domain.HumanMessage
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		s.Session.Data = &v
	case domain.SessionTypeAgent:
		var v domain.AgentMessage
		if err := unmarshalWithDefaultM(data, &v); err != nil {
			return err
		}
		s.Session.Data = &v
	case domain.SessionTypeToolExecution:
		var v domain.ToolExecution
		if err := unmarshalWithDefaultM(data, &v); err != nil {
			return err
		}
		s.Session.Data = &v
	case domain.SessionTypeSystem:
		var v domain.SystemMessage
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		s.Session.Data = &v
	default:
		// Fallback to raw bson.M for unknown types
		var raw bson.M
		if err := unmarshalWithDefaultM(data, &raw); err != nil {
			return err
		}
		s.Session.Data = raw
	}
	return nil
}
