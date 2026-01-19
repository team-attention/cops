package domain

// SystemTranscriptSubtype represents the subtype of a system transcript entry.
type SystemTranscriptSubtype string

const (
	SystemTranscriptSubtypeTurnDuration SystemTranscriptSubtype = "turn_duration"
)

// SystemTranscript represents system-level metadata in the conversation tree.
type SystemTranscript struct {
	TreeNodeMeta `bson:",inline"`

	Subtype    SystemTranscriptSubtype `json:"subtype" bson:"subtype"`
	DurationMs int                     `json:"durationMs" bson:"durationMs"`
	IsMeta     bool                    `json:"isMeta,omitempty" bson:"isMeta,omitempty"`
}
