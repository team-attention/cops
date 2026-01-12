package mongoschema

// NOTE: Transcript documents are stored in the `events` collection.
// Use EventCollectionName from event.go for collection access.

// Transcript root-level fields (type discriminator)
// Naming pattern: Transcript<FieldName>Field
const (
	TranscriptTypeField = "type"
)

// TreeNodeMeta fields (embedded in user, assistant, system transcripts)
// Naming pattern: TreeNode<FieldName>Field
const (
	TreeNodeParentUUIDField  = "parentUuid"
	TreeNodeUUIDField        = "uuid"
	TreeNodeSessionIDField   = "sessionId"
	TreeNodeTimestampField   = "timestamp"
	TreeNodeVersionField     = "version"
	TreeNodeCwdField         = "cwd"
	TreeNodeGitBranchField   = "gitBranch"
	TreeNodeSlugField        = "slug"
	TreeNodeUserTypeField    = "userType"
	TreeNodeIsSidechainField = "isSidechain"
)

// UserTranscript-specific fields
// Naming pattern: UserTranscript<FieldName>Field
const (
	UserTranscriptMessageField                 = "message"
	UserTranscriptIsMetaField                  = "isMeta"
	UserTranscriptThinkingMetadataField        = "thinkingMetadata"
	UserTranscriptTodosField                   = "todos"
	UserTranscriptToolUseResultField           = "toolUseResult"
	UserTranscriptSourceToolAssistantUUIDField = "sourceToolAssistantUUID"
)

// UserTranscriptMessage fields (inside UserTranscript.Message)
// Naming pattern: UserMessage<FieldName>Field
const (
	UserMessageRoleField    = "role"
	UserMessageContentField = "content"
)

// ThinkingMetadata fields (inside UserTranscript.ThinkingMetadata)
// Naming pattern: ThinkingMetadata<FieldName>Field
const (
	ThinkingMetadataLevelField    = "level"
	ThinkingMetadataDisabledField = "disabled"
	ThinkingMetadataTriggersField = "triggers"
)

// Todo fields (inside UserTranscript.Todos array)
// Naming pattern: Todo<FieldName>Field
const (
	TodoContentField    = "content"
	TodoStatusField     = "status"
	TodoActiveFormField = "activeForm"
)

// ToolUseResult fields (inside UserTranscript.ToolUseResult)
// Naming pattern: ToolUseResult<FieldName>Field
const (
	ToolUseResultSuccessField     = "success"
	ToolUseResultCommandNameField = "commandName"
	ToolUseResultModelField       = "model"
)

// AssistantTranscript-specific fields
// Naming pattern: AssistantTranscript<FieldName>Field
const (
	AssistantTranscriptRequestIDField = "requestId"
	AssistantTranscriptMessageField   = "message"
)

// AssistantTranscriptMessage fields (inside AssistantTranscript.Message)
// Naming pattern: AssistantMessage<FieldName>Field
const (
	AssistantMessageModelField        = "model"
	AssistantMessageIDField           = "id"
	AssistantMessageTypeField         = "type"
	AssistantMessageRoleField         = "role"
	AssistantMessageContentField      = "content"
	AssistantMessageStopReasonField   = "stopReason"
	AssistantMessageStopSequenceField = "stopSequence"
	AssistantMessageUsageField        = "usage"
)

// AssistantUsage fields (inside AssistantTranscriptMessage.Usage)
// Naming pattern: AssistantUsage<FieldName>Field
const (
	AssistantUsageInputTokensField              = "inputTokens"
	AssistantUsageOutputTokensField             = "outputTokens"
	AssistantUsageCacheCreationInputTokensField = "cacheCreationInputTokens"
	AssistantUsageCacheReadInputTokensField     = "cacheReadInputTokens"
	AssistantUsageCacheCreationField            = "cacheCreation"
	AssistantUsageServiceTierField              = "serviceTier"
)

// CacheCreation fields (inside AssistantUsage.CacheCreation)
// Naming pattern: CacheCreation<FieldName>Field
const (
	CacheCreationEphemeral5MInputTokensField = "ephemeral5MInputTokens"
	CacheCreationEphemeral1HInputTokensField = "ephemeral1HInputTokens"
)

// SystemTranscript-specific fields
// Naming pattern: SystemTranscript<FieldName>Field
const (
	SystemTranscriptSubtypeField    = "subtype"
	SystemTranscriptDurationMsField = "durationMs"
	SystemTranscriptIsMetaField     = "isMeta"
)

// SummaryTranscript fields
// Naming pattern: SummaryTranscript<FieldName>Field
const (
	SummaryTranscriptSummaryField  = "summary"
	SummaryTranscriptLeafUUIDField = "leafUuid"
)

// FileHistorySnapshotTranscript fields
// Naming pattern: FileHistorySnapshot<FieldName>Field
const (
	FileHistorySnapshotMessageIDField        = "messageId"
	FileHistorySnapshotSnapshotField         = "snapshot"
	FileHistorySnapshotIsSnapshotUpdateField = "isSnapshotUpdate"
)

// FileSnapshot fields (inside FileHistorySnapshotTranscript.Snapshot)
// Naming pattern: FileSnapshot<FieldName>Field
const (
	FileSnapshotMessageIDField          = "messageId"
	FileSnapshotTrackedFileBackupsField = "trackedFileBackups"
	FileSnapshotTimestampField          = "timestamp"
)

// FileBackup fields (inside FileSnapshot.TrackedFileBackups map values)
// Naming pattern: FileBackup<FieldName>Field
const (
	FileBackupBackupFileNameField = "backupFileName"
	FileBackupVersionField        = "version"
	FileBackupBackupTimeField     = "backupTime"
)

// Composite field paths for MongoDB queries (dot notation for nested fields)
// These are constructed by joining field constants to make relationships explicit
const (
	// AssistantTranscript.Message.Usage nested paths
	AssistantMessageUsageInputTokensPath              = AssistantTranscriptMessageField + "." + AssistantMessageUsageField + "." + AssistantUsageInputTokensField
	AssistantMessageUsageOutputTokensPath             = AssistantTranscriptMessageField + "." + AssistantMessageUsageField + "." + AssistantUsageOutputTokensField
	AssistantMessageUsageCacheCreationInputTokensPath = AssistantTranscriptMessageField + "." + AssistantMessageUsageField + "." + AssistantUsageCacheCreationInputTokensField
	AssistantMessageUsageCacheReadInputTokensPath     = AssistantTranscriptMessageField + "." + AssistantMessageUsageField + "." + AssistantUsageCacheReadInputTokensField

	// UserTranscript.ThinkingMetadata nested paths
	UserTranscriptThinkingMetadataLevelPath    = UserTranscriptThinkingMetadataField + "." + ThinkingMetadataLevelField
	UserTranscriptThinkingMetadataDisabledPath = UserTranscriptThinkingMetadataField + "." + ThinkingMetadataDisabledField

	// FileHistorySnapshotTranscript.Snapshot nested paths
	FileHistorySnapshotSnapshotMessageIDPath          = FileHistorySnapshotSnapshotField + "." + FileSnapshotMessageIDField
	FileHistorySnapshotSnapshotTimestampPath          = FileHistorySnapshotSnapshotField + "." + FileSnapshotTimestampField
	FileHistorySnapshotSnapshotTrackedFileBackupsPath = FileHistorySnapshotSnapshotField + "." + FileSnapshotTrackedFileBackupsField
)
