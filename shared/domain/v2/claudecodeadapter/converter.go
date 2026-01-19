package claudecodeadapter

import (
	"time"

	"github.com/team-attention/cops/shared/domain"
	v2 "github.com/team-attention/cops/shared/domain/v2"
)

// adaptUser converts UserTranscript to HumanMessage and/or ToolExecution.
//
// v1 UserTranscript can contain:
// - text/image content -> HumanMessage
// - tool_result content -> ToolExecution (result part)
//
// If it has both, returns multiple transcripts.
func (a *Adapter) adaptUser(u *domain.UserTranscript) ([]*v2.Transcript, error) {
	var results []*v2.Transcript

	humanContent, toolResults := a.splitUserContent(u)

	// Create HumanMessage if there's text/image content
	if len(humanContent) > 0 {
		human := &v2.HumanMessage{
			TreeNodeMeta: convertTreeNodeMeta(&u.TreeNodeMeta),
			Content:      humanContent,
			IsMeta:       u.IsMeta,
			Todos:        convertTodos(u.Todos),
		}
		results = append(results, &v2.Transcript{
			Type: v2.TranscriptTypeHuman,
			Data: human,
		})
	}

	// Create ToolExecution for each tool_result
	for _, tr := range toolResults {
		results = append(results, &v2.Transcript{
			Type: v2.TranscriptTypeToolExecution,
			Data: tr,
		})
	}

	return results, nil
}

// splitUserContent separates user content into human content and tool results.
func (a *Adapter) splitUserContent(u *domain.UserTranscript) ([]*v2.HumanContentBlock, []*v2.ToolExecution) {
	var humanBlocks []*v2.HumanContentBlock
	var toolExecs []*v2.ToolExecution

	// Handle string content (simple text message)
	if text, ok := u.Message.Content.(string); ok {
		humanBlocks = append(humanBlocks, &v2.HumanContentBlock{
			Type: v2.HumanContentBlockTypeText,
			Text: &text,
		})
		return humanBlocks, toolExecs
	}

	// Handle array content
	blocks, ok := u.Message.Content.([]any)
	if !ok {
		return humanBlocks, toolExecs
	}

	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := blockMap["type"].(string)

		switch blockType {
		case "text":
			text, _ := blockMap["text"].(string)
			humanBlocks = append(humanBlocks, &v2.HumanContentBlock{
				Type: v2.HumanContentBlockTypeText,
				Text: &text,
			})

		case "image":
			if source, ok := blockMap["source"].(map[string]any); ok {
				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)
				humanBlocks = append(humanBlocks, &v2.HumanContentBlock{
					Type: v2.HumanContentBlockTypeImage,
					Image: &v2.ImageData{
						MediaType: mediaType,
						Data:      data,
					},
				})
			}

		case "tool_result":
			toolUseID, _ := blockMap["tool_use_id"].(string)
			content := blockMap["content"]

			// Determine result status
			status := v2.ToolResultStatusSuccess
			var errMsg *string
			if isError, ok := blockMap["is_error"].(bool); ok && isError {
				status = v2.ToolResultStatusError
				if errStr, ok := content.(string); ok {
					errMsg = &errStr
				}
			}

			sourceAgentUUID := ""
			if u.SourceToolAssistantUUID != nil {
				sourceAgentUUID = *u.SourceToolAssistantUUID
			}

			toolExec := &v2.ToolExecution{
				TreeNodeMeta:    convertTreeNodeMeta(&u.TreeNodeMeta),
				ID:              toolUseID,
				ToolName:        "", // Will be filled from assistant message when correlating
				Input:           nil,
				SourceAgentUUID: sourceAgentUUID,
				Result: &v2.ToolResult{
					Status:  status,
					Content: content,
					Error:   errMsg,
				},
			}
			toolExecs = append(toolExecs, toolExec)
		}
	}

	return humanBlocks, toolExecs
}

// adaptAssistant converts AssistantTranscript to AgentMessage and ToolExecutions.
//
// v1 AssistantTranscript contains:
// - text/thinking content -> AgentMessage
// - tool_use content -> ToolExecution (call part)
func (a *Adapter) adaptAssistant(ast *domain.AssistantTranscript) ([]*v2.Transcript, error) {
	var results []*v2.Transcript
	var agentContent []*v2.AgentContentBlock
	var toolExecs []*v2.ToolExecution

	for _, block := range ast.Message.Content {
		switch block.Type {
		case "text":
			if block.Text != nil {
				agentContent = append(agentContent, &v2.AgentContentBlock{
					Type: v2.AgentContentBlockTypeText,
					Text: block.Text,
				})
			}

		case "thinking":
			if block.Thinking != nil {
				agentContent = append(agentContent, &v2.AgentContentBlock{
					Type: v2.AgentContentBlockTypeThinking,
					Thinking: &v2.ThinkingBlock{
						Content:   *block.Thinking,
						Signature: block.Signature,
					},
				})
			}

		case "tool_use":
			toolUseID := ""
			if block.ID != nil {
				toolUseID = *block.ID
			}
			toolName := ""
			if block.Name != nil {
				toolName = *block.Name
			}

			// Add reference in agent content
			agentContent = append(agentContent, &v2.AgentContentBlock{
				Type: v2.AgentContentBlockTypeToolCallRef,
				ToolCallRef: &v2.ToolCallReference{
					ToolExecutionID: toolUseID,
					ToolName:        toolName,
				},
			})

			// Create ToolExecution (call part only, result comes from user message)
			toolExec := &v2.ToolExecution{
				TreeNodeMeta:    convertTreeNodeMeta(&ast.TreeNodeMeta),
				ID:              toolUseID,
				ToolName:        toolName,
				Input:           block.Input,
				SourceAgentUUID: ast.UUID,
				// Result will be nil until correlated with user message tool_result
			}
			toolExecs = append(toolExecs, toolExec)
		}
	}

	// Create AgentMessage
	agent := &v2.AgentMessage{
		TreeNodeMeta: convertTreeNodeMeta(&ast.TreeNodeMeta),
		Provider:     "anthropic", // Claude Code uses Anthropic
		Model:        ast.Message.Model,
		RequestID:    ast.RequestID,
		Content:      agentContent,
		StopReason:   ast.Message.StopReason,
		Usage:        convertUsage(ast.Message.Usage),
	}
	results = append(results, &v2.Transcript{
		Type: v2.TranscriptTypeAgent,
		Data: agent,
	})

	// Add ToolExecutions
	for _, toolExec := range toolExecs {
		results = append(results, &v2.Transcript{
			Type: v2.TranscriptTypeToolExecution,
			Data: toolExec,
		})
	}

	return results, nil
}

// adaptSystem converts SystemTranscript to SystemMessage.
func (a *Adapter) adaptSystem(s *domain.SystemTranscript) ([]*v2.Transcript, error) {
	sys := &v2.SystemMessage{
		TreeNodeMeta: convertTreeNodeMeta(&s.TreeNodeMeta),
		Subtype:      v2.SystemMessageSubtypeTurnDuration,
		IsMeta:       s.IsMeta,
		TurnDuration: &v2.TurnDurationData{
			DurationMs: s.DurationMs,
		},
	}

	return []*v2.Transcript{{
		Type: v2.TranscriptTypeSystem,
		Data: sys,
	}}, nil
}

// adaptSummary converts SummaryTranscript to SystemMessage.
func (a *Adapter) adaptSummary(s *domain.SummaryTranscript) ([]*v2.Transcript, error) {
	sys := &v2.SystemMessage{
		TreeNodeMeta: v2.TreeNodeMeta{
			UUID:      s.LeafUUID + "-summary",
			Timestamp: time.Now(),
		},
		Subtype: v2.SystemMessageSubtypeSummary,
		Summary: &v2.SummaryData{
			Summary:  s.Summary,
			LeafUUID: s.LeafUUID,
		},
	}

	return []*v2.Transcript{{
		Type: v2.TranscriptTypeSystem,
		Data: sys,
	}}, nil
}

// adaptFileSnapshot converts FileHistorySnapshotTranscript to SystemMessage.
func (a *Adapter) adaptFileSnapshot(f *domain.FileHistorySnapshotTranscript) ([]*v2.Transcript, error) {
	backups := make(map[string]*v2.FileBackup)
	for path, backup := range f.Snapshot.TrackedFileBackups {
		backups[path] = &v2.FileBackup{
			BackupFileName: backup.BackupFileName,
			Version:        backup.Version,
			BackupTime:     backup.BackupTime,
		}
	}

	sys := &v2.SystemMessage{
		TreeNodeMeta: v2.TreeNodeMeta{
			UUID:      f.MessageID + "-snapshot",
			Timestamp: f.Snapshot.Timestamp,
		},
		Subtype: v2.SystemMessageSubtypeFileSnapshot,
		FileSnapshot: &v2.FileSnapshotData{
			MessageID:          f.MessageID,
			TrackedFileBackups: backups,
			SnapshotTimestamp:  f.Snapshot.Timestamp,
			IsSnapshotUpdate:   f.IsSnapshotUpdate,
		},
	}

	return []*v2.Transcript{{
		Type: v2.TranscriptTypeSystem,
		Data: sys,
	}}, nil
}

// adaptProgress converts ProgressTranscript to Progress.
func (a *Adapter) adaptProgress(p *domain.ProgressTranscript) ([]*v2.Transcript, error) {
	toolExecID := ""
	if p.ToolUseID != nil {
		toolExecID = *p.ToolUseID
	}
	parentToolExecID := ""
	if p.ParentToolUseID != nil {
		parentToolExecID = *p.ParentToolUseID
	}

	progress := &v2.Progress{
		TreeNodeMeta:          convertTreeNodeMeta(&p.TreeNodeMeta),
		ToolExecutionID:       toolExecID,
		ParentToolExecutionID: parentToolExecID,
		Data: v2.ProgressData{
			Type:               v2.ProgressType(p.Data.Type),
			Message:            p.Data.Message,
			NormalizedMessages: p.Data.NormalizedMessages,
			Prompt:             p.Data.Prompt,
			AgentID:            p.Data.AgentID,
		},
	}

	return []*v2.Transcript{{
		Type: v2.TranscriptTypeProgress,
		Data: progress,
	}}, nil
}

// convertTreeNodeMeta converts v1 TreeNodeMeta to v2.
func convertTreeNodeMeta(m *domain.TreeNodeMeta) v2.TreeNodeMeta {
	return v2.TreeNodeMeta{
		ParentUUID:  m.ParentUUID,
		UUID:        m.UUID,
		SessionID:   m.SessionID,
		Timestamp:   m.Timestamp,
		Version:     m.Version,
		CWD:         m.CWD,
		GitBranch:   m.GitBranch,
		Slug:        m.Slug,
		UserType:    m.UserType,
		IsSidechain: m.IsSidechain,
	}
}

// convertUsage converts v1 AssistantUsage to v2 TokenUsage.
func convertUsage(u *domain.AssistantUsage) *v2.TokenUsage {
	if u == nil {
		return nil
	}
	return &v2.TokenUsage{
		InputTokens:             u.InputTokens,
		OutputTokens:            u.OutputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:    u.CacheReadInputTokens,
		ServiceTier:             u.ServiceTier,
	}
}

// convertTodos converts v1 Todo slice to v2.
func convertTodos(todos []*domain.Todo) []*v2.Todo {
	if todos == nil {
		return nil
	}
	result := make([]*v2.Todo, len(todos))
	for i, t := range todos {
		result[i] = &v2.Todo{
			Content:    t.Content,
			Status:     t.Status,
			ActiveForm: t.ActiveForm,
		}
	}
	return result
}
