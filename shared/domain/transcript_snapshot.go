package domain

import "time"

// FileHistorySnapshotTranscript represents a file change tracking entry.
// Note: Independent structure - does NOT embed TreeNodeMeta.
type FileHistorySnapshotTranscript struct {
	MessageID        string       `json:"messageId" bson:"messageId"`
	Snapshot         FileSnapshot `json:"snapshot" bson:"snapshot"`
	IsSnapshotUpdate bool         `json:"isSnapshotUpdate" bson:"isSnapshotUpdate"`
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
