package mongoschema

import (
	"bytes"
	"time"

	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// unmarshalWithDefaultM unmarshals BSON data with DefaultDocumentM enabled.
// This ensures any/interface{} fields decode documents as map[string]interface{}
// instead of primitive.D, which is required for proper type assertions.
func unmarshalWithDefaultM(data []byte, v any) error {
	dec := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(data)))
	dec.DefaultDocumentM()
	return dec.Decode(v)
}

// EventCollectionName is the MongoDB collection name for transcript/event documents.
const EventCollectionName = "events"

// Event collection field names (aliases for backward compatibility).
// These constants are used by code that queries the events collection directly.
const (
	EventProjectIDField = "projectId"
)

// Transcript root-level fields (type discriminator)
// Naming pattern: Transcript<FieldName>Field
const (
	TranscriptTypeField      = "type"
	TranscriptUserIDField    = "userId"
	TranscriptProjectIDField = "projectId"
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

// ProgressTranscript-specific fields
// Naming pattern: ProgressTranscript<FieldName>Field
const (
	ProgressTranscriptToolUseIDField       = "toolUseID"
	ProgressTranscriptParentToolUseIDField = "parentToolUseID"
	ProgressTranscriptDataField            = "data"
)

// ProgressData fields (inside ProgressTranscript.Data)
// Naming pattern: ProgressData<FieldName>Field
const (
	ProgressDataTypeField               = "type"
	ProgressDataMessageField            = "message"
	ProgressDataNormalizedMessagesField = "normalizedMessages"
	ProgressDataPromptField             = "prompt"
	ProgressDataAgentIDField            = "agentId"
	ProgressDataHookEventField          = "hookEvent"
	ProgressDataHookNameField           = "hookName"
	ProgressDataCommandField            = "command"
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

// =============================================================================
// MongoDB Schema Structs
// =============================================================================

// Transcript is the MongoDB schema wrapper for domain.Transcript.
// Handles polymorphic Data field via MarshalBSON/UnmarshalBSON.
type Transcript struct {
	Type      domain.TranscriptType `bson:"type"`
	ProjectID bson.ObjectID         `bson:"projectId"`
	UserID    bson.ObjectID         `bson:"userId"`
	Data      any                   `bson:"-"` // Handled via MarshalBSON/UnmarshalBSON
}

// UnmarshalBSON implements bson.Unmarshaler for polymorphic deserialization.
func (t *Transcript) UnmarshalBSON(data []byte) error {
	// 1. Extract type, projectId, and userId first
	var base struct {
		Type      domain.TranscriptType `bson:"type"`
		ProjectID bson.ObjectID         `bson:"projectId"`
		UserID    bson.ObjectID         `bson:"userId"`
	}
	if err := bson.Unmarshal(data, &base); err != nil {
		return err
	}
	t.Type = base.Type
	t.ProjectID = base.ProjectID
	t.UserID = base.UserID

	// 2. Unmarshal to appropriate type based on discriminator
	switch t.Type {
	case domain.TranscriptTypeAssistant:
		var v AssistantTranscript
		// Use unmarshalWithDefaultM for types with any/interface{} fields
		// (AssistantContentBlock.Input is map[string]any)
		if err := unmarshalWithDefaultM(data, &v); err != nil {
			return err
		}
		t.Data = &v
	case domain.TranscriptTypeUser:
		var v UserTranscript
		// Use unmarshalWithDefaultM for types with any/interface{} fields
		// (UserMessage.Content and ToolUseResult are any)
		if err := unmarshalWithDefaultM(data, &v); err != nil {
			return err
		}
		t.Data = &v
	case domain.TranscriptTypeSystem:
		var v SystemTranscript
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		t.Data = &v
	case domain.TranscriptTypeSummary:
		var v SummaryTranscript
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		t.Data = &v
	case domain.TranscriptTypeFileHistorySnapshot:
		var v FileHistorySnapshotTranscript
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		t.Data = &v
	case domain.TranscriptTypeProgress:
		var v ProgressTranscript
		if err := bson.Unmarshal(data, &v); err != nil {
			return err
		}
		t.Data = &v
	default:
		// Fallback to raw bson.M for unknown types
		var raw bson.M
		if err := unmarshalWithDefaultM(data, &raw); err != nil {
			return err
		}
		t.Data = raw
	}
	return nil
}

// MarshalBSON implements bson.Marshaler for polymorphic serialization.
func (t Transcript) MarshalBSON() ([]byte, error) {
	// Marshal Data first, then add type and projectId
	dataBytes, err := bson.Marshal(t.Data)
	if err != nil {
		return nil, err
	}

	var doc bson.M
	if err := bson.Unmarshal(dataBytes, &doc); err != nil {
		return nil, err
	}

	doc[TranscriptTypeField] = string(t.Type)
	doc[TranscriptProjectIDField] = t.ProjectID
	doc[TranscriptUserIDField] = t.UserID

	return bson.Marshal(doc)
}

// ToDomain converts mongoschema.Transcript to domain.Transcript.
func (t *Transcript) ToDomain() *domain.Transcript {
	result := &domain.Transcript{Type: t.Type}

	switch data := t.Data.(type) {
	case *AssistantTranscript:
		result.Data = data.ToDomain()
	case *UserTranscript:
		result.Data = data.ToDomain()
	case *SystemTranscript:
		result.Data = data.ToDomain()
	case *SummaryTranscript:
		result.Data = data.ToDomain()
	case *FileHistorySnapshotTranscript:
		result.Data = data.ToDomain()
	case *ProgressTranscript:
		result.Data = data.ToDomain()
	default:
		result.Data = data
	}

	return result
}

// FromDomain converts domain.Transcript to mongoschema.Transcript.
func (t *Transcript) FromDomain(projectID, userID bson.ObjectID, d *domain.Transcript) {
	t.Type = d.Type
	t.ProjectID = projectID
	t.UserID = userID
	// Data is kept as-is (domain types have BSON tags)
	t.Data = d.Data
}

// =============================================================================
// TreeNodeMeta - Common fields for user/assistant/system transcripts
// =============================================================================

// TreeNodeMeta contains common fields for conversation tree nodes.
type TreeNodeMeta struct {
	ParentUUID  *string   `bson:"parentUuid,omitempty"`
	UUID        string    `bson:"uuid"`
	SessionID   string    `bson:"sessionId"`
	Timestamp   time.Time `bson:"timestamp"`
	Version     string    `bson:"version"`
	Cwd         string    `bson:"cwd,omitempty"`
	GitBranch   string    `bson:"gitBranch,omitempty"`
	Slug        string    `bson:"slug,omitempty"`
	UserType    string    `bson:"userType,omitempty"`
	IsSidechain bool      `bson:"isSidechain"`
}

// ToDomain converts mongoschema.TreeNodeMeta to domain.TreeNodeMeta.
func (t *TreeNodeMeta) ToDomain() domain.TreeNodeMeta {
	return domain.TreeNodeMeta{
		ParentUUID:  t.ParentUUID,
		UUID:        t.UUID,
		SessionID:   t.SessionID,
		Timestamp:   t.Timestamp,
		Version:     t.Version,
		CWD:         t.Cwd,
		GitBranch:   t.GitBranch,
		Slug:        t.Slug,
		UserType:    t.UserType,
		IsSidechain: t.IsSidechain,
	}
}

// =============================================================================
// AssistantTranscript
// =============================================================================

// AssistantTranscript is the MongoDB schema for domain.AssistantTranscript.
type AssistantTranscript struct {
	TreeNodeMeta `bson:",inline"`
	RequestID    string           `bson:"requestId"`
	Message      AssistantMessage `bson:"message"`
}

// AssistantMessage is the MongoDB schema for domain.AssistantTranscriptMessage.
type AssistantMessage struct {
	Model        string                   `bson:"model"`
	ID           string                   `bson:"id"`
	Type         string                   `bson:"type"`
	Role         string                   `bson:"role"`
	Content      []*AssistantContentBlock `bson:"content"`
	StopReason   *string                  `bson:"stopReason,omitempty"`
	StopSequence *string                  `bson:"stopSequence,omitempty"`
	Usage        *AssistantUsage          `bson:"usage,omitempty"`
}

// AssistantContentBlock is the MongoDB schema for domain.AssistantContentBlock.
type AssistantContentBlock struct {
	Type      string         `bson:"type"`
	Thinking  *string        `bson:"thinking,omitempty"`
	Signature *string        `bson:"signature,omitempty"`
	Text      *string        `bson:"text,omitempty"`
	ID        *string        `bson:"id,omitempty"`
	Name      *string        `bson:"name,omitempty"`
	Input     map[string]any `bson:"input,omitempty"`
}

// AssistantUsage is the MongoDB schema for domain.AssistantUsage.
type AssistantUsage struct {
	InputTokens              int            `bson:"inputTokens"`
	OutputTokens             int            `bson:"outputTokens"`
	CacheCreationInputTokens int            `bson:"cacheCreationInputTokens"`
	CacheReadInputTokens     int            `bson:"cacheReadInputTokens"`
	CacheCreation            *CacheCreation `bson:"cacheCreation,omitempty"`
	ServiceTier              string         `bson:"serviceTier,omitempty"`
}

// CacheCreation is the MongoDB schema for domain.CacheCreation.
type CacheCreation struct {
	Ephemeral5mInputTokens int `bson:"ephemeral5MInputTokens"`
	Ephemeral1hInputTokens int `bson:"ephemeral1HInputTokens"`
}

// ToDomain converts mongoschema.AssistantTranscript to domain.AssistantTranscript.
func (t *AssistantTranscript) ToDomain() *domain.AssistantTranscript {
	return &domain.AssistantTranscript{
		TreeNodeMeta: t.TreeNodeMeta.ToDomain(),
		RequestID:    t.RequestID,
		Message:      t.Message.ToDomain(),
	}
}

// ToDomain converts mongoschema.AssistantMessage to domain.AssistantTranscriptMessage.
func (m *AssistantMessage) ToDomain() domain.AssistantTranscriptMessage {
	content := make([]*domain.AssistantContentBlock, len(m.Content))
	for i, c := range m.Content {
		content[i] = c.ToDomain()
	}

	return domain.AssistantTranscriptMessage{
		Model:        m.Model,
		ID:           m.ID,
		Type:         m.Type,
		Role:         m.Role,
		Content:      content,
		StopReason:   m.StopReason,
		StopSequence: m.StopSequence,
		Usage:        m.Usage.ToDomain(),
	}
}

// ToDomain converts mongoschema.AssistantContentBlock to domain.AssistantContentBlock.
func (c *AssistantContentBlock) ToDomain() *domain.AssistantContentBlock {
	return &domain.AssistantContentBlock{
		Type:      c.Type,
		Thinking:  c.Thinking,
		Signature: c.Signature,
		Text:      c.Text,
		ID:        c.ID,
		Name:      c.Name,
		Input:     c.Input,
	}
}

// ToDomain converts mongoschema.AssistantUsage to domain.AssistantUsage.
func (u *AssistantUsage) ToDomain() *domain.AssistantUsage {
	if u == nil {
		return nil
	}
	return &domain.AssistantUsage{
		InputTokens:              u.InputTokens,
		OutputTokens:             u.OutputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
		CacheCreation:            u.CacheCreation.ToDomain(),
		ServiceTier:              u.ServiceTier,
	}
}

// ToDomain converts mongoschema.CacheCreation to domain.CacheCreation.
func (c *CacheCreation) ToDomain() *domain.CacheCreation {
	if c == nil {
		return nil
	}
	return &domain.CacheCreation{
		Ephemeral5mInputTokens: c.Ephemeral5mInputTokens,
		Ephemeral1hInputTokens: c.Ephemeral1hInputTokens,
	}
}

// =============================================================================
// UserTranscript
// =============================================================================

// UserTranscript is the MongoDB schema for domain.UserTranscript.
type UserTranscript struct {
	TreeNodeMeta            `bson:",inline"`
	Message                 UserMessage       `bson:"message"`
	IsMeta                  bool              `bson:"isMeta,omitempty"`
	ThinkingMetadata        *ThinkingMetadata `bson:"thinkingMetadata,omitempty"`
	Todos                   []*Todo           `bson:"todos,omitempty"`
	ToolUseResult           any               `bson:"toolUseResult,omitempty"`
	SourceToolAssistantUUID *string           `bson:"sourceToolAssistantUUID,omitempty"`
}

// UserMessage is the MongoDB schema for domain.UserTranscriptMessage.
type UserMessage struct {
	Role    string `bson:"role"`
	Content any    `bson:"content"`
}

// ThinkingMetadata is the MongoDB schema for domain.ThinkingMetadata.
type ThinkingMetadata struct {
	Level    string             `bson:"level"`
	Disabled bool               `bson:"disabled"`
	Triggers []*ThinkingTrigger `bson:"triggers,omitempty"`
}

// ThinkingTrigger is the MongoDB schema for domain.ThinkingTrigger.
type ThinkingTrigger struct {
	Start int    `bson:"start"`
	End   int    `bson:"end"`
	Text  string `bson:"text"`
}

// Todo is the MongoDB schema for domain.Todo.
type Todo struct {
	Content    string `bson:"content"`
	Status     string `bson:"status"`
	ActiveForm string `bson:"activeForm"`
}

// ToDomain converts mongoschema.UserTranscript to domain.UserTranscript.
func (t *UserTranscript) ToDomain() *domain.UserTranscript {
	return &domain.UserTranscript{
		TreeNodeMeta:            t.TreeNodeMeta.ToDomain(),
		Message:                 t.Message.ToDomain(),
		IsMeta:                  t.IsMeta,
		ThinkingMetadata:        t.ThinkingMetadata.ToDomain(),
		Todos:                   todosToDoamin(t.Todos),
		ToolUseResult:           t.ToolUseResult,
		SourceToolAssistantUUID: t.SourceToolAssistantUUID,
	}
}

// ToDomain converts mongoschema.UserMessage to domain.UserTranscriptMessage.
func (m *UserMessage) ToDomain() domain.UserTranscriptMessage {
	return domain.UserTranscriptMessage{
		Role:    m.Role,
		Content: m.Content,
	}
}

// ToDomain converts mongoschema.ThinkingMetadata to domain.ThinkingMetadata.
func (t *ThinkingMetadata) ToDomain() *domain.ThinkingMetadata {
	if t == nil {
		return nil
	}
	triggers := make([]*domain.ThinkingTrigger, len(t.Triggers))
	for i, tr := range t.Triggers {
		triggers[i] = tr.ToDomain()
	}
	return &domain.ThinkingMetadata{
		Level:    t.Level,
		Disabled: t.Disabled,
		Triggers: triggers,
	}
}

// ToDomain converts mongoschema.ThinkingTrigger to domain.ThinkingTrigger.
func (t *ThinkingTrigger) ToDomain() *domain.ThinkingTrigger {
	if t == nil {
		return nil
	}
	return &domain.ThinkingTrigger{
		Start: t.Start,
		End:   t.End,
		Text:  t.Text,
	}
}

func todosToDoamin(todos []*Todo) []*domain.Todo {
	if todos == nil {
		return nil
	}
	result := make([]*domain.Todo, len(todos))
	for i, t := range todos {
		result[i] = t.ToDomain()
	}
	return result
}

// ToDomain converts mongoschema.Todo to domain.Todo.
func (t *Todo) ToDomain() *domain.Todo {
	if t == nil {
		return nil
	}
	return &domain.Todo{
		Content:    t.Content,
		Status:     t.Status,
		ActiveForm: t.ActiveForm,
	}
}

// =============================================================================
// SystemTranscript
// =============================================================================

// SystemTranscript is the MongoDB schema for domain.SystemTranscript.
type SystemTranscript struct {
	TreeNodeMeta `bson:",inline"`
	Subtype      domain.SystemTranscriptSubtype `bson:"subtype"`
	DurationMs   int                            `bson:"durationMs"`
	IsMeta       bool                           `bson:"isMeta,omitempty"`
}

// ToDomain converts mongoschema.SystemTranscript to domain.SystemTranscript.
func (t *SystemTranscript) ToDomain() *domain.SystemTranscript {
	return &domain.SystemTranscript{
		TreeNodeMeta: t.TreeNodeMeta.ToDomain(),
		Subtype:      t.Subtype,
		DurationMs:   t.DurationMs,
		IsMeta:       t.IsMeta,
	}
}

// =============================================================================
// SummaryTranscript
// =============================================================================

// SummaryTranscript is the MongoDB schema for domain.SummaryTranscript.
type SummaryTranscript struct {
	Summary  string `bson:"summary"`
	LeafUUID string `bson:"leafUuid"`
}

// ToDomain converts mongoschema.SummaryTranscript to domain.SummaryTranscript.
func (t *SummaryTranscript) ToDomain() *domain.SummaryTranscript {
	return &domain.SummaryTranscript{
		Summary:  t.Summary,
		LeafUUID: t.LeafUUID,
	}
}

// =============================================================================
// FileHistorySnapshotTranscript
// =============================================================================

// FileHistorySnapshotTranscript is the MongoDB schema for domain.FileHistorySnapshotTranscript.
type FileHistorySnapshotTranscript struct {
	MessageID        string       `bson:"messageId"`
	Snapshot         FileSnapshot `bson:"snapshot"`
	IsSnapshotUpdate bool         `bson:"isSnapshotUpdate"`
}

// FileSnapshot is the MongoDB schema for domain.FileSnapshot.
type FileSnapshot struct {
	MessageID          string                 `bson:"messageId"`
	TrackedFileBackups map[string]*FileBackup `bson:"trackedFileBackups"`
	Timestamp          time.Time              `bson:"timestamp"`
}

// FileBackup is the MongoDB schema for domain.FileBackup.
type FileBackup struct {
	BackupFileName *string   `bson:"backupFileName,omitempty"`
	Version        int       `bson:"version"`
	BackupTime     time.Time `bson:"backupTime"`
}

// ToDomain converts mongoschema.FileHistorySnapshotTranscript to domain.FileHistorySnapshotTranscript.
func (t *FileHistorySnapshotTranscript) ToDomain() *domain.FileHistorySnapshotTranscript {
	return &domain.FileHistorySnapshotTranscript{
		MessageID:        t.MessageID,
		Snapshot:         t.Snapshot.ToDomain(),
		IsSnapshotUpdate: t.IsSnapshotUpdate,
	}
}

// ToDomain converts mongoschema.FileSnapshot to domain.FileSnapshot.
func (s *FileSnapshot) ToDomain() domain.FileSnapshot {
	backups := make(map[string]*domain.FileBackup, len(s.TrackedFileBackups))
	for k, v := range s.TrackedFileBackups {
		backups[k] = v.ToDomain()
	}
	return domain.FileSnapshot{
		MessageID:          s.MessageID,
		TrackedFileBackups: backups,
		Timestamp:          s.Timestamp,
	}
}

// ToDomain converts mongoschema.FileBackup to domain.FileBackup.
func (b *FileBackup) ToDomain() *domain.FileBackup {
	if b == nil {
		return nil
	}
	return &domain.FileBackup{
		BackupFileName: b.BackupFileName,
		Version:        b.Version,
		BackupTime:     b.BackupTime,
	}
}

// =============================================================================
// ProgressTranscript
// =============================================================================

// ProgressTranscript is the MongoDB schema for domain.ProgressTranscript.
type ProgressTranscript struct {
	TreeNodeMeta    `bson:",inline"`
	ToolUseID       *string      `bson:"toolUseID,omitempty"`
	ParentToolUseID *string      `bson:"parentToolUseID,omitempty"`
	Data            ProgressData `bson:"data"`
}

// ProgressData is the MongoDB schema for domain.ProgressData.
type ProgressData struct {
	Type               domain.ProgressDataType `bson:"type"`
	Message            map[string]any          `bson:"message,omitempty"`
	NormalizedMessages []map[string]any        `bson:"normalizedMessages,omitempty"`
	Prompt             *string                 `bson:"prompt,omitempty"`
	AgentID            *string                 `bson:"agentId,omitempty"`
	// Hook progress fields
	HookEvent *string `bson:"hookEvent,omitempty"`
	HookName  *string `bson:"hookName,omitempty"`
	Command   *string `bson:"command,omitempty"`
}

// ToDomain converts mongoschema.ProgressTranscript to domain.ProgressTranscript.
func (t *ProgressTranscript) ToDomain() *domain.ProgressTranscript {
	return &domain.ProgressTranscript{
		TreeNodeMeta:    t.TreeNodeMeta.ToDomain(),
		ToolUseID:       t.ToolUseID,
		ParentToolUseID: t.ParentToolUseID,
		Data:            t.Data.ToDomain(),
	}
}

// ToDomain converts mongoschema.ProgressData to domain.ProgressData.
func (d *ProgressData) ToDomain() domain.ProgressData {
	return domain.ProgressData{
		Type:               d.Type,
		Message:            d.Message,
		NormalizedMessages: d.NormalizedMessages,
		Prompt:             d.Prompt,
		AgentID:            d.AgentID,
		HookEvent:          d.HookEvent,
		HookName:           d.HookName,
		Command:            d.Command,
	}
}
