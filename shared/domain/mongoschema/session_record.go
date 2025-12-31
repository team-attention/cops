package mongoschema

const (
	// RecordCollectionName is the MongoDB collection name for records.
	// Name unchanged for continuity.
	RecordCollectionName = "sessionRecords"
)

// Record-level fields (root level of document)
// Naming pattern: Record<FieldName>Field
const (
	RecordIDField              = "_id"
	RecordTypeField            = "type"
	RecordParentUUIDField      = "parentUuid"
	RecordIsSidechainField     = "isSidechain"
	RecordUserTypeField        = "userType"
	RecordSessionIDField       = "sessionId"
	RecordVersionField         = "version"
	RecordGitBranchField       = "gitBranch"
	RecordUUIDField            = "uuid"
	RecordTimestampField       = "timestamp"
	RecordProjectIDField       = "projectId"
	RecordIsMetaField          = "isMeta"
	RecordMessageField         = "message"
	RecordThinkingMetadataField = "thinkingMetadata"
	RecordTodosField           = "todos"
	RecordRequestIDField       = "requestId"
	RecordMessageIDField       = "messageId"       // For FileHistorySnapshotRecord
	RecordSnapshotField        = "snapshot"
	RecordIsSnapshotUpdateField = "isSnapshotUpdate"
)

// Message-level fields (inside Record.Message object)
// Naming pattern: Message<FieldName>Field
const (
	MessageModelField      = "model"
	MessageIDField         = "id"
	MessageTypeField       = "type"
	MessageRoleField       = "role"
	MessageContentField    = "content"
	MessageStopReasonField = "stopReason"
	MessageStopSequenceField = "stopSequence"
	MessageUsageField      = "usage"
)

// Usage-level fields (inside Message.Usage object)
// Naming pattern: Usage<FieldName>Field
const (
	UsageInputTokensField         = "inputTokens"
	UsageOutputTokensField        = "outputTokens"
	UsageCacheCreationTokensField = "cacheCreationInputTokens"
	UsageCacheReadTokensField     = "cacheReadInputTokens"
	UsageServiceTierField         = "serviceTier"
)

// ThinkingMetadata-level fields (inside Record.ThinkingMetadata object)
// Naming pattern: ThinkingMetadata<FieldName>Field
const (
	ThinkingMetadataLevelField    = "level"
	ThinkingMetadataDisabledField = "disabled"
	ThinkingMetadataTriggersField = "triggers"
)

// Snapshot-level fields (inside Record.Snapshot object)
// Naming pattern: Snapshot<FieldName>Field
const (
	SnapshotMessageIDField          = "messageId"
	SnapshotTrackedFileBackupsField = "trackedFileBackups"
)

// Composite field paths for MongoDB queries (dot notation for nested fields)
// These are constructed by joining field constants to make the relationship explicit
const (
	// Message.Usage nested paths
	MessageUsageInputTokensPath       = RecordMessageField + "." + MessageUsageField + "." + UsageInputTokensField
	MessageUsageOutputTokensPath      = RecordMessageField + "." + MessageUsageField + "." + UsageOutputTokensField
	MessageUsageCacheReadTokensPath   = RecordMessageField + "." + MessageUsageField + "." + UsageCacheReadTokensField
	MessageUsageCacheCreationTokensPath = RecordMessageField + "." + MessageUsageField + "." + UsageCacheCreationTokensField
)
