package geminicliadapter

import (
	session "github.com/team-attention/cops/shared/domain/v2"
)

// adaptUserMessage converts a Gemini user message to HumanMessage.
func (a *Adapter) adaptUserMessage(msg *GeminiMessage, sessionID string) []*session.Session {
	content := msg.Content
	human := &session.HumanMessage{
		TreeNodeMeta: session.TreeNodeMeta{
			UUID:      msg.ID,
			SessionID: sessionID,
			Timestamp: msg.Timestamp,
		},
		Content: []*session.HumanContentBlock{
			{
				Type: session.HumanContentBlockTypeText,
				Text: &content,
			},
		},
	}

	return []*session.Session{{
		Type: session.SessionTypeHuman,
		Data: human,
	}}
}

// adaptGeminiMessage converts a Gemini agent message to AgentMessage and ToolExecutions.
func (a *Adapter) adaptGeminiMessage(msg *GeminiMessage, sessionID string) []*session.Session {
	var results []*session.Session
	var agentContent []*session.AgentContentBlock

	// Add thinking blocks
	agentContent = append(agentContent, a.convertThoughts(msg.Thoughts)...)

	// Add text content if present
	if msg.Content != "" {
		content := msg.Content
		agentContent = append(agentContent, &session.AgentContentBlock{
			Type: session.AgentContentBlockTypeText,
			Text: &content,
		})
	}

	// Add tool call references
	var toolExecs []*session.ToolExecution
	for _, tc := range msg.ToolCalls {
		// Add reference in agent content
		agentContent = append(agentContent, &session.AgentContentBlock{
			Type: session.AgentContentBlockTypeToolCallRef,
			ToolCallRef: &session.ToolCallReference{
				ToolExecutionID: tc.ID,
				ToolName:        tc.Name,
			},
		})

		// Create ToolExecution
		toolExec := a.adaptToolCall(tc, msg.ID, sessionID)
		toolExecs = append(toolExecs, toolExec)
	}

	// Create AgentMessage
	agent := &session.AgentMessage{
		TreeNodeMeta: session.TreeNodeMeta{
			UUID:      msg.ID,
			SessionID: sessionID,
			Timestamp: msg.Timestamp,
		},
		Provider: "google",
		Model:    msg.Model,
		Content:  agentContent,
		Usage:    a.convertTokens(msg.Tokens),
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

	return results
}

// adaptInfoMessage converts a Gemini info message to SystemMessage.
func (a *Adapter) adaptInfoMessage(msg *GeminiMessage, sessionID string) []*session.Session {
	// Info messages are system-level metadata in Gemini CLI.
	// Map to SystemMessage with the info subtype.
	sys := &session.SystemMessage{
		TreeNodeMeta: session.TreeNodeMeta{
			UUID:      msg.ID,
			SessionID: sessionID,
			Timestamp: msg.Timestamp,
		},
		Subtype: session.SystemMessageSubtypeInfo,
	}

	return []*session.Session{{
		Type: session.SessionTypeSystem,
		Data: sys,
	}}
}

// adaptToolCall converts a Gemini tool call to ToolExecution.
// Unlike Claude Code where tool_use and tool_result are separate,
// Gemini combines call and result in a single toolCalls entry.
func (a *Adapter) adaptToolCall(tc *GeminiToolCall, sourceAgentUUID, sessionID string) *session.ToolExecution {
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
	if status == session.ToolResultStatusError {
		if tc.ResultDisplay != "" {
			errMsg = &tc.ResultDisplay
		}
	}

	return &session.ToolExecution{
		TreeNodeMeta: session.TreeNodeMeta{
			UUID:      tc.ID,
			SessionID: sessionID,
			Timestamp: tc.Timestamp,
		},
		ID:              tc.ID,
		ToolName:        tc.Name,
		Input:           tc.Args,
		SourceAgentUUID: sourceAgentUUID,
		Result: &session.ToolResult{
			Status:  status,
			Content: resultContent,
			Error:   errMsg,
		},
	}
}

// convertThoughts converts Gemini thoughts to agent content blocks.
func (a *Adapter) convertThoughts(thoughts []*GeminiThought) []*session.AgentContentBlock {
	if len(thoughts) == 0 {
		return nil
	}

	var blocks []*session.AgentContentBlock
	for _, thought := range thoughts {
		// Combine subject and description into thinking content
		content := thought.Subject
		if thought.Description != "" {
			content += "\n\n" + thought.Description
		}

		blocks = append(blocks, &session.AgentContentBlock{
			Type: session.AgentContentBlockTypeThinking,
			Thinking: &session.ThinkingBlock{
				Content: content,
			},
		})
	}

	return blocks
}

// convertTokens converts Gemini token usage to v2 TokenUsage.
func (a *Adapter) convertTokens(tokens *GeminiTokens) *session.TokenUsage {
	if tokens == nil {
		return nil
	}

	return &session.TokenUsage{
		InputTokens:  tokens.Input,
		OutputTokens: tokens.Output,
		// Gemini's "cached" maps to cache read tokens
		CacheReadInputTokens: tokens.Cached,
	}
}

// convertToolStatus converts Gemini tool status to v2 ToolResultStatus.
func convertToolStatus(status string) session.ToolResultStatus {
	switch status {
	case "success":
		return session.ToolResultStatusSuccess
	case "error":
		return session.ToolResultStatusError
	case "cancelled":
		return session.ToolResultStatusSkipped
	default:
		return session.ToolResultStatusSuccess
	}
}
