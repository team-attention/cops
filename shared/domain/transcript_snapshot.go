package domain

import "time"

// FileHistorySnapshotTranscript represents a file change tracking entry.
// Records file backup state at specific points during code modifications.
//
// IMPORTANT: Does NOT embed TreeNodeMeta - no sessionId, uuid, or timestamp at root level.
// References the associated message via MessageID.
// Contains Snapshot.Timestamp but no root-level timestamp field.
// Should be EXCLUDED from session aggregation queries.
type FileHistorySnapshotTranscript struct {
	// MessageID references the user/assistant message that triggered this snapshot.
	MessageID string `json:"messageId" bson:"messageId"`
	// Snapshot contains the file backup state.
	Snapshot FileSnapshot `json:"snapshot" bson:"snapshot"`
	// IsSnapshotUpdate indicates if this is an update to existing snapshot vs new snapshot.
	IsSnapshotUpdate bool `json:"isSnapshotUpdate" bson:"isSnapshotUpdate"`
}

// FileSnapshot represents the file backup state at a point in time.
type FileSnapshot struct {
	MessageID          string                 `json:"messageId" bson:"messageId"`
	TrackedFileBackups map[string]*FileBackup `json:"trackedFileBackups" bson:"trackedFileBackups"`
	Timestamp          time.Time              `json:"timestamp" bson:"timestamp"`
}

// FileBackup represents backup info for a single file.
type FileBackup struct {
	BackupFileName *string   `json:"backupFileName,omitempty" bson:"backupFileName,omitempty"`
	Version        int       `json:"version" bson:"version"`
	BackupTime     time.Time `json:"backupTime" bson:"backupTime"`
}
