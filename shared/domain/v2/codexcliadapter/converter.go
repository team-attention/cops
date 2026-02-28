package codexcliadapter

import (
	"strings"
	"time"

	session "github.com/team-attention/cops/shared/domain/v2"
)

// adaptEventMsg converts an event_msg entry to v2 sessions.
func (a *Adapter) adaptEventMsg(payload *EventMsgPayload, timestamp time.Time) []*session.Session {
	switch payload.Type {
	case "user_message":
		msg := payload.Message
		human := &session.HumanMessage{
			TreeNodeMeta: a.buildTreeNodeMeta(timestamp),
			Content: []*session.HumanContentBlock{
				{
					Type: session.HumanContentBlockTypeText,
					Text: &msg,
				},
			},
		}
		return []*session.Session{{
			Type: session.SessionTypeHuman,
			Data: human,
		}}

	case "agent_message":
		msg := payload.Message
		agent := &session.AgentMessage{
			TreeNodeMeta: a.buildTreeNodeMeta(timestamp),
			Provider:     a.resolveProvider(),
			Model:        a.cached.model,
			Content: []*session.AgentContentBlock{
				{
					Type: session.AgentContentBlockTypeText,
					Text: &msg,
				},
			},
		}
		return []*session.Session{{
			Type: session.SessionTypeAgent,
			Data: agent,
		}}

	case "agent_reasoning":
		agent := &session.AgentMessage{
			TreeNodeMeta: a.buildTreeNodeMeta(timestamp),
			Provider:     a.resolveProvider(),
			Model:        a.cached.model,
			Content: []*session.AgentContentBlock{
				{
					Type: session.AgentContentBlockTypeThinking,
					Thinking: &session.ThinkingBlock{
						Content: payload.Text,
					},
				},
			},
		}
		return []*session.Session{{
			Type: session.SessionTypeAgent,
			Data: agent,
		}}

	case "token_count", "task_started", "task_complete":
		return nil

	default:
		return nil
	}
}

// adaptResponseItem converts a response_item entry to v2 sessions.
func (a *Adapter) adaptResponseItem(payload *ResponseItemPayload, timestamp time.Time) []*session.Session {
	if payload.Type == "reasoning" {
		var texts []string
		for _, s := range payload.Summary {
			if s.Text != "" {
				texts = append(texts, s.Text)
			}
		}
		concatenated := strings.Join(texts, "\n")
		if concatenated == "" {
			return nil
		}
		agent := &session.AgentMessage{
			TreeNodeMeta: a.buildTreeNodeMeta(timestamp),
			Provider:     a.resolveProvider(),
			Model:        a.cached.model,
			Content: []*session.AgentContentBlock{
				{
					Type: session.AgentContentBlockTypeThinking,
					Thinking: &session.ThinkingBlock{
						Content: concatenated,
					},
				},
			},
		}
		return []*session.Session{{
			Type: session.SessionTypeAgent,
			Data: agent,
		}}
	}

	if payload.Type == "message" {
		if payload.Role != nil && *payload.Role == "assistant" {
			var blocks []*session.AgentContentBlock
			for _, block := range payload.Content {
				if block.Type == "output_text" {
					text := block.Text
					blocks = append(blocks, &session.AgentContentBlock{
						Type: session.AgentContentBlockTypeText,
						Text: &text,
					})
				}
			}
			if len(blocks) == 0 {
				return nil
			}
			agent := &session.AgentMessage{
				TreeNodeMeta: a.buildTreeNodeMeta(timestamp),
				Provider:     a.resolveProvider(),
				Model:        a.cached.model,
				Content:      blocks,
			}
			return []*session.Session{{
				Type: session.SessionTypeAgent,
				Data: agent,
			}}
		}

		if payload.Role != nil && (*payload.Role == "user" || *payload.Role == "developer") {
			return nil
		}
	}

	return nil
}

// resolveProvider returns the model provider string for AgentMessage.Provider.
// Uses the cached modelProvider from session_meta if available, defaults to DefaultModelProvider.
func (a *Adapter) resolveProvider() string {
	if a.cached.modelProvider != "" {
		return a.cached.modelProvider
	}
	return DefaultModelProvider
}
