package codexcliadapter

import "time"

// SessionMetaPayload is the payload for type="session_meta".
// Contains session-level metadata.
//
// v2 Mapping:
//   - ID             -> TreeNodeMeta.SessionID
//   - CWD            -> Project association (absolute path matching)
//   - ModelProvider   -> AgentMessage.Provider
type SessionMetaPayload struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	CWD           string    `json:"cwd"`
	Originator    *string   `json:"originator,omitempty"`     // e.g., "codex_cli_rs", "codex_vscode"; optional, not used for v2 mapping
	CLIVersion    *string   `json:"cli_version,omitempty"`    // optional, not used for v2 mapping
	Source        any       `json:"source,omitempty"`          // string "cli" or object {"subagent":{...}}; not used for v2 mapping
	ModelProvider *string   `json:"model_provider,omitempty"` // e.g., "openai"; absent in older versions (pre-v0.106.0)
}

// EventMsgPayload is the payload for type="event_msg".
// The inner Type field further discriminates the event kind.
//
// v2 Mapping by inner Type:
//   - "user_message"     -> HumanMessage
//   - "agent_message"    -> AgentMessage (text content)
//   - "agent_reasoning"  -> AgentContentBlock (type=thinking)
//   - "token_count"      -> Not mapped (lifecycle/metrics event)
//   - "task_started"     -> Not mapped (internal lifecycle event)
//   - "task_complete"    -> Not mapped (internal lifecycle event)
type EventMsgPayload struct {
	Type    string `json:"type"` // "user_message", "agent_message", "agent_reasoning", "token_count", "task_started", "task_complete"
	Message string `json:"message,omitempty"`
	Text    string `json:"text,omitempty"` // For agent_reasoning

	// For token_count
	Info *TokenCountInfo `json:"info,omitempty"`

	// For task_started/task_complete
	TurnID                *string `json:"turn_id,omitempty"`
	LastAgentMessage      *string `json:"last_agent_message,omitempty"`
	ModelContextWindow    *int    `json:"model_context_window,omitempty"`
	CollaborationModeKind *string `json:"collaboration_mode_kind,omitempty"`
}

// TokenCountInfo contains token usage info from token_count events.
//
// v2 Mapping:
//   - TotalTokenUsage -> TokenUsage
//   - ModelContextWindow -> not mapped
type TokenCountInfo struct {
	TotalTokenUsage    *TokenUsageDetail `json:"total_token_usage,omitempty"`
	LastTokenUsage     *TokenUsageDetail `json:"last_token_usage,omitempty"`
	ModelContextWindow *int              `json:"model_context_window,omitempty"`
}

// TokenUsageDetail contains detailed token counts.
//
// v2 Mapping:
//   - InputTokens           -> TokenUsage.InputTokens
//   - OutputTokens          -> TokenUsage.OutputTokens
//   - CachedInputTokens     -> TokenUsage.CacheReadInputTokens
//   - ReasoningOutputTokens -> not mapped (Codex-specific, may be absent in non-reasoning models)
type TokenUsageDetail struct {
	InputTokens           int  `json:"input_tokens"`
	CachedInputTokens     int  `json:"cached_input_tokens"`
	OutputTokens          int  `json:"output_tokens"`
	ReasoningOutputTokens *int `json:"reasoning_output_tokens,omitempty"`
	TotalTokens           int  `json:"total_tokens"`
}

// ResponseItemPayload is the payload for type="response_item".
// Contains structured message content from the model API.
//
// v2 Mapping:
//   - Role="user"/"developer" -> Ignored (system/developer prompts, not user input)
//   - Role="assistant"        -> AgentMessage (text via output_text content blocks)
//   - Type="reasoning"        -> AgentContentBlock (type=thinking, via Summary)
type ResponseItemPayload struct {
	Type    string                  `json:"type"` // "message" or "reasoning"
	Role    *string                 `json:"role,omitempty"` // "user", "developer", "assistant"
	Content []*ResponseContentBlock `json:"content,omitempty"`
	Phase   *string                 `json:"phase,omitempty"` // e.g., "final_answer"

	// For type="reasoning"
	Summary          []*ReasoningSummary `json:"summary,omitempty"`
	EncryptedContent *string             `json:"encrypted_content,omitempty"` // Not mapped
}

// ResponseContentBlock represents a content block within a response_item.
type ResponseContentBlock struct {
	Type string `json:"type"` // "input_text", "output_text"
	Text string `json:"text"`
}

// ReasoningSummary represents a summary entry in a reasoning response_item.
type ReasoningSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// TurnContextPayload is the payload for type="turn_context".
// Contains model and collaboration context for a turn.
//
// v2 Mapping:
//   - Model  -> AgentMessage.Model (cached by adapter, applied to subsequent sessions)
//   - TurnID -> used for grouping entries within a turn
type TurnContextPayload struct {
	TurnID         string  `json:"turn_id"`
	CWD            string  `json:"cwd"`
	Model          string  `json:"model"`
	Personality    *string `json:"personality,omitempty"`
	Effort         *string `json:"effort,omitempty"`
	ApprovalPolicy *string `json:"approval_policy,omitempty"`
}
