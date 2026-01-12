package domain

// SummaryTranscript represents a conversation branch summary.
// Used for context management in long conversations.
// Note: Independent structure - does NOT embed TreeNodeMeta.
type SummaryTranscript struct {
	Summary  string `json:"summary" bson:"summary"`
	LeafUUID string `json:"leafUuid" bson:"leafUuid"`
}
