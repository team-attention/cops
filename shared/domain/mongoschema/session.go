package mongoschema

// SessionsCollectionName is the MongoDB collection name for session documents.
const SessionsCollectionName = "sessions"

// Session collection field names.
// Naming pattern: Session<FieldName>Field
const (
	SessionIDField        = "_id"
	SessionProjectIDField = "projectId"
	SessionUserIDField    = "userId"
	SessionSessionIDField = "sessionId"
	SessionTypeField      = "type"
	SessionTimestampField = "timestamp"
)
