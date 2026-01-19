package domain

// SummaryTranscript represents a conversation branch summary.
// Used for context management in long conversations (auto-summarization).
//
// IMPORTANT: Does NOT embed TreeNodeMeta - no sessionId, uuid, or timestamp.
// References a conversation tree node via LeafUUID.
// Should be EXCLUDED from session aggregation queries.
type SummaryTranscript struct {
	// Summary is the AI-generated summary text of the conversation branch.
	Summary string `json:"summary" bson:"summary"`
	// LeafUUID references the conversation tree node (user/assistant/system uuid) this summary covers.
	LeafUUID string `json:"leafUuid" bson:"leafUuid"`
}
