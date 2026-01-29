package mongoschema

// ResumeTokensCollectionName is the MongoDB collection name for resume token documents.
const ResumeTokensCollectionName = "resume_tokens"

// ResumeToken collection field names.
// Naming pattern: ResumeToken<FieldName>Field
const (
	ResumeTokenKeyField       = "key"
	ResumeTokenTokenField     = "token"
	ResumeTokenUpdatedAtField = "updatedAt"
)
