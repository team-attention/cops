package opencodeadapter

import (
	"time"

	session "github.com/team-attention/cops/shared/domain/v2"
)

// createdTime returns the created_at timestamp as time.Time.
// Returns zero time if msg is nil or CreatedAt is 0.
func createdTime(msg *OpenCodeMessage) time.Time {
	if msg == nil || msg.CreatedAt == 0 {
		return time.Time{}
	}
	return time.Unix(msg.CreatedAt, 0)
}

// adaptUserMessage converts an OpenCode user message to HumanMessage.
// The parts parameter is pre-parsed by AdaptMessage.
func (a *Adapter) adaptUserMessage(msg *OpenCodeMessage, parts []*OpenCodePart) []*session.Session {
	meta := buildTreeNodeMeta(msg)

	var humanContent []*session.HumanContentBlock
	var toolExecs []*session.ToolExecution

	for _, part := range parts {
		switch part.Type {
		case "text":
			if part.Text != nil {
				text := *part.Text
				humanContent = append(humanContent, &session.HumanContentBlock{
					Type: session.HumanContentBlockTypeText,
					Text: &text,
				})
			}
		case "tool-result":
			toolExec := &session.ToolExecution{
				TreeNodeMeta: session.TreeNodeMeta{
					SessionID: msg.SessionID,
					Timestamp: meta.Timestamp,
					Provider:  "open_code",
				},
				Result: &session.ToolResult{
					Status:  session.ToolResultStatusSuccess,
					Content: part.Result,
				},
			}
			if part.ToolID != nil {
				toolExec.ID = *part.ToolID
				toolExec.TreeNodeMeta.UUID = *part.ToolID
			}
			if part.Name != nil {
				toolExec.ToolName = *part.Name
			}
			toolExecs = append(toolExecs, toolExec)
		}
	}

	human := &session.HumanMessage{
		TreeNodeMeta: meta,
		Content:      humanContent,
	}

	results := []*session.Session{{
		Type: session.SessionTypeHuman,
		Data: human,
	}}

	for _, toolExec := range toolExecs {
		results = append(results, &session.Session{
			Type: session.SessionTypeToolExecution,
			Data: toolExec,
		})
	}

	return results
}

// adaptAssistantMessage converts an OpenCode assistant message to AgentMessage and ToolExecutions.
// The parts parameter is pre-parsed by AdaptMessage.
func (a *Adapter) adaptAssistantMessage(msg *OpenCodeMessage, parts []*OpenCodePart) []*session.Session {
	meta := buildTreeNodeMeta(msg)

	var agentContent []*session.AgentContentBlock
	var toolExecs []*session.ToolExecution

	for _, part := range parts {
		switch part.Type {
		case "text":
			if part.Text != nil {
				text := *part.Text
				agentContent = append(agentContent, &session.AgentContentBlock{
					Type: session.AgentContentBlockTypeText,
					Text: &text,
				})
			}
		case "tool-invocation":
			// Skip partial-call (incomplete tool invocations)
			if part.State != nil && *part.State == "partial-call" {
				continue
			}

			// Add ToolCallRef to agent content
			var toolID, toolName string
			if part.ToolID != nil {
				toolID = *part.ToolID
			}
			if part.Name != nil {
				toolName = *part.Name
			}

			agentContent = append(agentContent, &session.AgentContentBlock{
				Type: session.AgentContentBlockTypeToolCallRef,
				ToolCallRef: &session.ToolCallReference{
					ToolExecutionID: toolID,
					ToolName:        toolName,
				},
			})

			// Create ToolExecution
			toolExec := &session.ToolExecution{
				TreeNodeMeta: session.TreeNodeMeta{
					UUID:      toolID,
					SessionID: msg.SessionID,
					Timestamp: meta.Timestamp,
					Provider:  "open_code",
				},
				ID:              toolID,
				ToolName:        toolName,
				Input:           part.Args,
				SourceAgentUUID: msg.ID,
			}

			// If state="result", attach the result
			if part.State != nil && *part.State == "result" {
				toolExec.Result = &session.ToolResult{
					Status:  session.ToolResultStatusSuccess,
					Content: part.Result,
				}
			}

			toolExecs = append(toolExecs, toolExec)
		}
	}

	agent := &session.AgentMessage{
		TreeNodeMeta: meta,
		Provider:     "open_code",
		Model:        msg.Model,
		Content:      agentContent,
		Usage:        nil, // Token usage not available in the messages table schema
	}

	results := []*session.Session{{
		Type: session.SessionTypeAgent,
		Data: agent,
	}}

	for _, toolExec := range toolExecs {
		results = append(results, &session.Session{
			Type: session.SessionTypeToolExecution,
			Data: toolExec,
		})
	}

	return results
}

// buildTreeNodeMeta creates a TreeNodeMeta from an OpenCode message.
func buildTreeNodeMeta(msg *OpenCodeMessage) session.TreeNodeMeta {
	return session.TreeNodeMeta{
		UUID:      msg.ID,
		SessionID: msg.SessionID,
		Provider:  "open_code",
		Timestamp: createdTime(msg),
	}
}
