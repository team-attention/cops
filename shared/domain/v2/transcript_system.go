package v2

import "time"

// SystemMessageSubtype represents the subtype of a system message.
type SystemMessageSubtype string

const (
	SystemMessageSubtypeTurnDuration SystemMessageSubtype = "turn_duration"
	SystemMessageSubtypeSummary      SystemMessageSubtype = "summary"
	SystemMessageSubtypeFileSnapshot SystemMessageSubtype = "file_snapshot"
	SystemMessageSubtypeInfo         SystemMessageSubtype = "info" // Provider-agnostic informational message
)

// SystemMessage represents system-level metadata in the conversation.
// Uses Subtype discriminator to determine which data field is populated.
//
// This type consolidates v1's separate transcript types:
//   - system (turn_duration)  -> SystemMessage with Subtype="turn_duration"
//   - summary                 -> SystemMessage with Subtype="summary"
//   - file-history-snapshot   -> SystemMessage with Subtype="file_snapshot"
type SystemMessage struct {
	// TreeNodeMeta is embedded for turn_duration subtype.
	// For summary and file_snapshot, UUID/Timestamp may be generated or derived.
	TreeNodeMeta `bson:",inline"`

	Subtype SystemMessageSubtype `json:"subtype" bson:"subtype"`
	IsMeta  bool                 `json:"isMeta,omitempty" bson:"isMeta,omitempty"`

	// For subtype="turn_duration"
	TurnDuration *TurnDurationData `json:"turnDuration,omitempty" bson:"turnDuration,omitempty"`

	// For subtype="summary"
	Summary *SummaryData `json:"summary,omitempty" bson:"summary,omitempty"`

	// For subtype="file_snapshot"
	FileSnapshot *FileSnapshotData `json:"fileSnapshot,omitempty" bson:"fileSnapshot,omitempty"`
}

// TurnDurationData contains turn duration metrics.
type TurnDurationData struct {
	DurationMs int `json:"durationMs" bson:"durationMs"`
}

// SummaryData contains conversation branch summary.
// Used for context management in long conversations (auto-summarization).
type SummaryData struct {
	// Summary is the AI-generated summary text of the conversation branch.
	Summary string `json:"summary" bson:"summary"`

	// LeafUUID references the conversation tree node this summary covers.
	LeafUUID string `json:"leafUuid" bson:"leafUuid"`
}

// FileSnapshotData contains file change tracking information.
// Records file backup state at specific points during code modifications.
type FileSnapshotData struct {
	// MessageID references the message that triggered this snapshot.
	MessageID string `json:"messageId" bson:"messageId"`

	// TrackedFileBackups contains backup info keyed by file path.
	TrackedFileBackups map[string]*FileBackup `json:"trackedFileBackups" bson:"trackedFileBackups"`

	// SnapshotTimestamp is when the snapshot was taken.
	SnapshotTimestamp time.Time `json:"snapshotTimestamp" bson:"snapshotTimestamp"`

	// IsSnapshotUpdate indicates if this is an update to existing snapshot vs new snapshot.
	IsSnapshotUpdate bool `json:"isSnapshotUpdate,omitempty" bson:"isSnapshotUpdate,omitempty"`
}

// FileBackup represents backup info for a single file.
type FileBackup struct {
	BackupFileName *string   `json:"backupFileName,omitempty" bson:"backupFileName,omitempty"`
	Version        int       `json:"version" bson:"version"`
	BackupTime     time.Time `json:"backupTime" bson:"backupTime"`
}
