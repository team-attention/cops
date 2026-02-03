// Package claudecodeadapter provides conversion from Claude Code v1 transcript format to v2 session format.
//
// Claude Code produces JSONL logs with Claude API-specific structures:
//   - User messages contain tool_result blocks
//   - Assistant messages contain tool_use blocks
//
// This adapter converts these to the provider-agnostic v2 session format where:
//   - ToolExecution is an independent type containing both call and result
//   - HumanMessage contains only text/image content
//   - AgentMessage contains text/thinking and tool call references
package claudecodeadapter

import (
	"github.com/team-attention/cops/shared/domain"
	session "github.com/team-attention/cops/shared/domain/v2"
)

// Adapter converts Claude Code v1 transcripts to v2 session format.
type Adapter struct{}

// NewAdapter creates a new ClaudeCodeAdapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Adapt converts a v1 Transcript to v2 Session format.
// May return multiple v2 Sessions (e.g., assistant message splits into AgentMessage + ToolExecutions).
func (a *Adapter) Adapt(t *domain.Transcript) ([]*session.Session, error) {
	switch t.Type {
	case domain.TranscriptTypeUser:
		return a.adaptUser(t.Data.(*domain.UserTranscript))
	case domain.TranscriptTypeAssistant:
		return a.adaptAssistant(t.Data.(*domain.AssistantTranscript))
	case domain.TranscriptTypeSystem:
		return a.adaptSystem(t.Data.(*domain.SystemTranscript))
	case domain.TranscriptTypeSummary:
		return a.adaptSummary(t.Data.(*domain.SummaryTranscript))
	case domain.TranscriptTypeFileHistorySnapshot:
		// Skip: file snapshots are internal backup state, not needed for dashboard
		return nil, nil
	case domain.TranscriptTypeProgress:
		// Skip: progress events are transient UI state, not needed for dashboard
		return nil, nil
	default:
		return nil, nil
	}
}

// AdaptBatch converts multiple v1 Transcripts to v2 Session format.
func (a *Adapter) AdaptBatch(transcripts []*domain.Transcript) ([]*session.Session, error) {
	var result []*session.Session
	for _, t := range transcripts {
		adapted, err := a.Adapt(t)
		if err != nil {
			return nil, err
		}
		result = append(result, adapted...)
	}
	return result, nil
}
