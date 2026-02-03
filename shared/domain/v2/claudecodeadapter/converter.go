package claudecodeadapter

import (
	"time"

	"github.com/team-attention/cops/shared/domain"
	session "github.com/team-attention/cops/shared/domain/v2"
)

// adaptUser converts UserTranscript to HumanMessage and/or ToolExecution.
//
// v1 UserTranscript can contain:
// - text/image content -> HumanMessage
// - tool_result content -> ToolExecution (result part)
//
// If it has both, returns multiple sessions.
func (a *Adapter) adaptUser(u *domain.UserTranscript) ([]*session.Session, error) {
	var results []*session.Session

	humanContent, toolResults := a.splitUserContent(u)

	// Create HumanMessage if there's text/image content
	if len(humanContent) > 0 {
		human := &session.HumanMessage{
			TreeNodeMeta: convertTreeNodeMeta(&u.TreeNodeMeta),
			Content:      humanContent,
			IsMeta:       u.IsMeta,
			Todos:        convertTodos(u.Todos),
		}
		results = append(results, &session.Session{
			Type: session.SessionTypeHuman,
			Data: human,
		})
	}

	// Create ToolExecution for each tool_result
	for _, tr := range toolResults {
		results = append(results, &session.Session{
			Type: session.SessionTypeToolExecution,
			Data: tr,
		})
	}

	return results, nil
}

// splitUserContent separates user content into human content and tool results.
func (a *Adapter) splitUserContent(u *domain.UserTranscript) ([]*session.HumanContentBlock, []*session.ToolExecution) {
	var humanBlocks []*session.HumanContentBlock
	var toolExecs []*session.ToolExecution

	// Handle string content (simple text message)
	if text, ok := u.Message.Content.(string); ok {
		humanBlocks = append(humanBlocks, &session.HumanContentBlock{
			Type: session.HumanContentBlockTypeText,
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
			humanBlocks = append(humanBlocks, &session.HumanContentBlock{
				Type: session.HumanContentBlockTypeText,
				Text: &text,
			})

		case "image":
			if source, ok := blockMap["source"].(map[string]any); ok {
				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)
				humanBlocks = append(humanBlocks, &session.HumanContentBlock{
					Type: session.HumanContentBlockTypeImage,
					Image: &session.ImageData{
						MediaType: mediaType,
						Data:      data,
					},
				})
			}

		case "tool_result":
			toolUseID, _ := blockMap["tool_use_id"].(string)
			content := blockMap["content"]

			// Determine result status
			status := session.ToolResultStatusSuccess
			var errMsg *string
			if isError, ok := blockMap["is_error"].(bool); ok && isError {
				status = session.ToolResultStatusError
				if errStr, ok := content.(string); ok {
					errMsg = &errStr
				}
			}

			sourceAgentUUID := ""
			if u.SourceToolAssistantUUID != nil {
				sourceAgentUUID = *u.SourceToolAssistantUUID
			}

			toolExec := &session.ToolExecution{
				TreeNodeMeta:    convertTreeNodeMeta(&u.TreeNodeMeta),
				ID:              toolUseID,
				ToolName:        "", // Will be filled from assistant message when correlating
				Input:           nil,
				SourceAgentUUID: sourceAgentUUID,
				Metadata:        buildToolExecutionMetadata(u.TreeNodeMeta.AgentID),
				Result: &session.ToolResult{
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
func (a *Adapter) adaptAssistant(ast *domain.AssistantTranscript) ([]*session.Session, error) {
	var results []*session.Session
	var agentContent []*session.AgentContentBlock
	var toolExecs []*session.ToolExecution

	for _, block := range ast.Message.Content {
		switch block.Type {
		case "text":
			if block.Text != nil {
				agentContent = append(agentContent, &session.AgentContentBlock{
					Type: session.AgentContentBlockTypeText,
					Text: block.Text,
				})
			}

		case "thinking":
			if block.Thinking != nil {
				agentContent = append(agentContent, &session.AgentContentBlock{
					Type: session.AgentContentBlockTypeThinking,
					Thinking: &session.ThinkingBlock{
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
			agentContent = append(agentContent, &session.AgentContentBlock{
				Type: session.AgentContentBlockTypeToolCallRef,
				ToolCallRef: &session.ToolCallReference{
					ToolExecutionID: toolUseID,
					ToolName:        toolName,
				},
			})

			// Create ToolExecution (call part only, result comes from user message)
			toolExec := &session.ToolExecution{
				TreeNodeMeta:    convertTreeNodeMeta(&ast.TreeNodeMeta),
				ID:              toolUseID,
				ToolName:        toolName,
				Input:           block.Input,
				SourceAgentUUID: ast.UUID,
				Metadata:        buildToolExecutionMetadata(ast.TreeNodeMeta.AgentID),
				// Result will be nil until correlated with user message tool_result
			}
			toolExecs = append(toolExecs, toolExec)
		}
	}

	// Create AgentMessage
	agent := &session.AgentMessage{
		TreeNodeMeta: convertTreeNodeMeta(&ast.TreeNodeMeta),
		Provider:     "anthropic", // Claude Code uses Anthropic
		Model:        ast.Message.Model,
		RequestID:    ast.RequestID,
		Content:      agentContent,
		StopReason:   ast.Message.StopReason,
		Usage:        convertUsage(ast.Message.Usage),
	}
	results = append(results, &session.Session{
		Type: session.SessionTypeAgent,
		Data: agent,
	})

	// Add ToolExecutions
	for _, toolExec := range toolExecs {
		results = append(results, &session.Session{
			Type: session.SessionTypeToolExecution,
			Data: toolExec,
		})
	}

	return results, nil
}

// adaptSystem converts SystemTranscript to SystemMessage.
func (a *Adapter) adaptSystem(s *domain.SystemTranscript) ([]*session.Session, error) {
	sys := &session.SystemMessage{
		TreeNodeMeta: convertTreeNodeMeta(&s.TreeNodeMeta),
		Subtype:      session.SystemMessageSubtypeTurnDuration,
		IsMeta:       s.IsMeta,
		TurnDuration: &session.TurnDurationData{
			DurationMs: s.DurationMs,
		},
	}

	return []*session.Session{{
		Type: session.SessionTypeSystem,
		Data: sys,
	}}, nil
}

// adaptSummary converts SummaryTranscript to SystemMessage.
func (a *Adapter) adaptSummary(s *domain.SummaryTranscript) ([]*session.Session, error) {
	sys := &session.SystemMessage{
		TreeNodeMeta: session.TreeNodeMeta{
			UUID:      s.LeafUUID + "-summary",
			Timestamp: time.Now(),
		},
		Subtype: session.SystemMessageSubtypeSummary,
		Summary: &session.SummaryData{
			Summary:  s.Summary,
			LeafUUID: s.LeafUUID,
		},
	}

	return []*session.Session{{
		Type: session.SessionTypeSystem,
		Data: sys,
	}}, nil
}

// convertTreeNodeMeta converts v1 TreeNodeMeta to session.
func convertTreeNodeMeta(m *domain.TreeNodeMeta) session.TreeNodeMeta {
	return session.TreeNodeMeta{
		ParentUUID:  m.ParentUUID,
		UUID:        m.UUID,
		SessionID:   m.SessionID,
		AgentID:     m.AgentID,
		Timestamp:   m.Timestamp,
		Provider:    "claude_code",
		Version:     m.Version,
		GitBranch:   m.GitBranch,
		IsSidechain: m.IsSidechain,
	}
}

// convertUsage converts v1 AssistantUsage to session TokenUsage.
func convertUsage(u *domain.AssistantUsage) *session.TokenUsage {
	if u == nil {
		return nil
	}
	return &session.TokenUsage{
		InputTokens:             u.InputTokens,
		OutputTokens:            u.OutputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:    u.CacheReadInputTokens,
		ServiceTier:             u.ServiceTier,
	}
}

// convertTodos converts v1 Todo slice to session.
func convertTodos(todos []*domain.Todo) []*session.Todo {
	if todos == nil {
		return nil
	}
	result := make([]*session.Todo, len(todos))
	for i, t := range todos {
		result[i] = &session.Todo{
			Content:    t.Content,
			Status:     t.Status,
			ActiveForm: t.ActiveForm,
		}
	}
	return result
}

// buildToolExecutionMetadata creates ToolExecutionMetadata for SubAgent entries.
// Returns nil if agentID is nil or empty (Main session).
func buildToolExecutionMetadata(agentID *string) *session.ToolExecutionMetadata {
	if agentID == nil {
		return nil
	}
	if *agentID == "" {
		return nil
	}
	return &session.ToolExecutionMetadata{
		IsSubagent: true,
		SubagentID: *agentID,
	}
}
