package domain

import "time"

// ------- Type FileHistorySnapshot -------

type FileHistorySnapshotTrackedBackups struct {
	BackupFileName *string   `json:"backupFileName,omitempty" bson:"backupFileName,omitempty"`
	Version        int       `json:"version" bson:"version"`
	BackupTime     time.Time `json:"backupTime" bson:"backupTime"`
}

type FileHistorySnapshot struct {
	MessageID          string                                       `json:"messageId" bson:"messageId"`
	TrackedFileBackups map[string]FileHistorySnapshotTrackedBackups `json:"trackedFileBackups" bson:"trackedFileBackups"`
}

type FileHistorySnapshotRecord struct {
	MessageID        string              `json:"messageId" bson:"messageId"`
	Snapshot         FileHistorySnapshot `json:"snapshot" bson:"snapshot"`
	IsSnapshotUpdate bool                `json:"isSnapshotUpdate" bson:"isSnapshotUpdate"`
}
