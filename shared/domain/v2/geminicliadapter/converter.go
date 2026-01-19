package geminicliadapter

import (
	v2 "github.com/team-attention/cops/shared/domain/v2"
)

// adaptUserMessage converts a Gemini user message to HumanMessage.
func (a *Adapter) adaptUserMessage(msg *GeminiMessage, sessionID string) []*v2.Transcript {
	content := msg.Content
	human := &v2.HumanMessage{
		TreeNodeMeta: v2.TreeNodeMeta{
			UUID:      msg.ID,
			SessionID: sessionID,
			Timestamp: msg.Timestamp,
		},
		Content: []*v2.HumanContentBlock{
			{
				Type: v2.HumanContentBlockTypeText,
				Text: &content,
			},
		},
	}

	return []*v2.Transcript{{
		Type: v2.TranscriptTypeHuman,
		Data: human,
	}}
}

// adaptGeminiMessage converts a Gemini agent message to AgentMessage and ToolExecutions.
func (a *Adapter) adaptGeminiMessage(msg *GeminiMessage, sessionID string) []*v2.Transcript {
	var results []*v2.Transcript
	var agentContent []*v2.AgentContentBlock

	// Add thinking blocks
	agentContent = append(agentContent, a.convertThoughts(msg.Thoughts)...)

	// Add text content if present
	if msg.Content != "" {
		content := msg.Content
		agentContent = append(agentContent, &v2.AgentContentBlock{
			Type: v2.AgentContentBlockTypeText,
			Text: &content,
		})
	}

	// Add tool call references
	var toolExecs []*v2.ToolExecution
	for _, tc := range msg.ToolCalls {
		// Add reference in agent content
		agentContent = append(agentContent, &v2.AgentContentBlock{
			Type: v2.AgentContentBlockTypeToolCallRef,
			ToolCallRef: &v2.ToolCallReference{
				ToolExecutionID: tc.ID,
				ToolName:        tc.Name,
			},
		})

		// Create ToolExecution
		toolExec := a.adaptToolCall(tc, msg.ID, sessionID)
		toolExecs = append(toolExecs, toolExec)
	}

	// Create AgentMessage
	agent := &v2.AgentMessage{
		TreeNodeMeta: v2.TreeNodeMeta{
			UUID:      msg.ID,
			SessionID: sessionID,
			Timestamp: msg.Timestamp,
		},
		Provider: "google",
		Model:    msg.Model,
		Content:  agentContent,
		Usage:    a.convertTokens(msg.Tokens),
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

	return results
}

// adaptInfoMessage converts a Gemini info message to SystemMessage.
func (a *Adapter) adaptInfoMessage(msg *GeminiMessage, sessionID string) []*v2.Transcript {
	// Info messages are system-level metadata in Gemini CLI.
	// Map to SystemMessage with the info subtype.
	sys := &v2.SystemMessage{
		TreeNodeMeta: v2.TreeNodeMeta{
			UUID:      msg.ID,
			SessionID: sessionID,
			Timestamp: msg.Timestamp,
		},
		Subtype: v2.SystemMessageSubtypeInfo,
	}

	return []*v2.Transcript{{
		Type: v2.TranscriptTypeSystem,
		Data: sys,
	}}
}

// adaptToolCall converts a Gemini tool call to ToolExecution.
// Unlike Claude Code where tool_use and tool_result are separate,
// Gemini combines call and result in a single toolCalls entry.
func (a *Adapter) adaptToolCall(tc *GeminiToolCall, sourceAgentUUID, sessionID string) *v2.ToolExecution {
	// Determine result status
	status := convertToolStatus(tc.Status)

	// Extract result content
	var resultContent any
	var errMsg *string
	if len(tc.Result) > 0 {
		// If there's only one result, unwrap it
		if len(tc.Result) == 1 {
			resultContent = tc.Result[0]
		} else {
			resultContent = tc.Result
		}
	}

	// If status is error, try to extract error message
	if status == v2.ToolResultStatusError {
		if tc.ResultDisplay != "" {
			errMsg = &tc.ResultDisplay
		}
	}

	return &v2.ToolExecution{
		TreeNodeMeta: v2.TreeNodeMeta{
			UUID:      tc.ID,
			SessionID: sessionID,
			Timestamp: tc.Timestamp,
		},
		ID:              tc.ID,
		ToolName:        tc.Name,
		Input:           tc.Args,
		SourceAgentUUID: sourceAgentUUID,
		Result: &v2.ToolResult{
			Status:  status,
			Content: resultContent,
			Error:   errMsg,
		},
	}
}

// convertThoughts converts Gemini thoughts to agent content blocks.
func (a *Adapter) convertThoughts(thoughts []*GeminiThought) []*v2.AgentContentBlock {
	if len(thoughts) == 0 {
		return nil
	}

	var blocks []*v2.AgentContentBlock
	for _, thought := range thoughts {
		// Combine subject and description into thinking content
		content := thought.Subject
		if thought.Description != "" {
			content += "\n\n" + thought.Description
		}

		blocks = append(blocks, &v2.AgentContentBlock{
			Type: v2.AgentContentBlockTypeThinking,
			Thinking: &v2.ThinkingBlock{
				Content: content,
			},
		})
	}

	return blocks
}

// convertTokens converts Gemini token usage to v2 TokenUsage.
func (a *Adapter) convertTokens(tokens *GeminiTokens) *v2.TokenUsage {
	if tokens == nil {
		return nil
	}

	return &v2.TokenUsage{
		InputTokens:  tokens.Input,
		OutputTokens: tokens.Output,
		// Gemini's "cached" maps to cache read tokens
		CacheReadInputTokens: tokens.Cached,
	}
}

// convertToolStatus converts Gemini tool status to v2 ToolResultStatus.
func convertToolStatus(status string) v2.ToolResultStatus {
	switch status {
	case "success":
		return v2.ToolResultStatusSuccess
	case "error":
		return v2.ToolResultStatusError
	case "cancelled":
		return v2.ToolResultStatusSkipped
	default:
		return v2.ToolResultStatusSuccess
	}
}
