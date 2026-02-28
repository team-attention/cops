// Package codexcliadapter provides conversion from Codex CLI JSONL logs to v2 session format.
//
// # Codex CLI Log Location
//
// Codex CLI produces JSONL files at:
//
//	~/.codex/sessions/YYYY/MM/DD/rollout-{timestamp}-{uuid}.jsonl
//
// # Type Mapping: Codex CLI -> v2
//
// The following table shows how Codex CLI entry types map to v2 session types:
//
//	| Codex Entry Type            | v2 Type                              | Notes                                    |
//	|-----------------------------|--------------------------------------|------------------------------------------|
//	| event_msg (user_message)    | HumanMessage                         | User input text                          |
//	| event_msg (agent_message)   | AgentMessage (type=text)             | Agent text response                      |
//	| event_msg (agent_reasoning) | AgentMessage (type=thinking)         | Extended thinking blocks                 |
//	| event_msg (token_count)     | Not mapped                           | Lifecycle/metrics event                  |
//	| event_msg (task_started)    | Not mapped                           | Internal lifecycle event                 |
//	| event_msg (task_complete)   | Not mapped                           | Internal lifecycle event                 |
//	| response_item (assistant)   | AgentMessage (type=text)             | Structured assistant response            |
//	| response_item (reasoning)   | AgentMessage (type=thinking)         | Reasoning summary blocks                 |
//	| response_item (user/dev)    | Not mapped                           | System/developer prompts, not user input |
//	| session_meta                | Metadata (cached: SessionID, Provider)| Session-level metadata                  |
//	| turn_context                | Metadata (cached: Model)             | Turn-level model context                 |
//
// # Key Differences from Claude Code and Gemini CLI
//
// Unlike Claude Code (which receives pre-parsed domain.Transcript with SessionID embedded)
// and Gemini CLI (which receives a GeminiSession struct with SessionID at the root),
// the Codex adapter receives raw JSONL lines where metadata is spread across separate entries.
// The adapter is stateful: it caches session ID from session_meta and model from turn_context,
// applying them to all subsequently produced sessions.
package codexcliadapter

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	session "github.com/team-attention/cops/shared/domain/v2"
)

// ProviderName is the provider string used in TreeNodeMeta.Provider for all sessions
// produced by this adapter. Defined as a package-level constant for consistency with
// how ProviderCodexCLI is defined in api/internal/service/session/provider.go.
const ProviderName = "codex_cli"

// DefaultModelProvider is the default provider string used in AgentMessage.Provider
// when no model_provider has been cached from a session_meta entry.
const DefaultModelProvider = "openai"

// metadataTypes are the Codex CLI entry types that contain metadata (not content).
// These are processed first in AdaptBatch to populate cached state before content conversion.
var metadataTypes = map[string]bool{
	"session_meta": true,
	"turn_context": true,
}

// parsedEntry holds the pre-parsed fields from a single JSONL line.
// Used by AdaptBatch to avoid double JSON parsing across the two-pass processing.
type parsedEntry struct {
	entryType    string
	timestamp    time.Time
	payloadBytes []byte
}

// cachedMeta groups the metadata fields cached from session_meta and turn_context entries.
// Grouping these fields improves readability and makes state reset explicit.
type cachedMeta struct {
	sessionID     string // Cached from session_meta.ID
	modelProvider string // Cached from session_meta.ModelProvider
	model         string // Cached from turn_context.Model
}

// Adapter converts Codex CLI JSONL entries to v2 session format.
// It is stateful: session_meta and turn_context entries update internal
// cached state (sessionID, model, provider) which is applied to all
// subsequently produced sessions.
type Adapter struct {
	logger *slog.Logger
	cached cachedMeta
}

// NewAdapter creates a new Codex CLI adapter.
func NewAdapter(l *slog.Logger) *Adapter {
	return &Adapter{
		logger: l.With(slog.String("name", "codexcliadapter")),
	}
}

// AdaptEntry converts a single raw JSONL line to v2 Session records.
// Returns nil sessions for entries that don't map to v2 types (task_started, task_complete, developer/user response_items).
// Returns at most one session per entry. Each entry produces either zero or one session.
// Metadata entries (session_meta, turn_context) update the adapter's cached state and return nil sessions.
//
// The return type is []*session.Session (not *session.Session) to maintain API consistency
// with AdaptBatch. While each Codex entry is atomic (producing 0 or 1 sessions, unlike Claude
// entries which can expand to AgentMessage + ToolExecution(s)), using the same slice return type
// keeps the caller interface uniform.
func (a *Adapter) AdaptEntry(line string) ([]*session.Session, error) {
	if line == "" {
		return nil, nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSONL line: %w", err)
	}

	typeVal, ok := raw["type"]
	if !ok {
		return nil, nil
	}
	entryType, ok := typeVal.(string)
	if !ok {
		return nil, nil
	}

	timestamp := extractTimestamp(raw)

	payloadVal, ok := raw["payload"]
	if !ok {
		return nil, nil
	}
	payloadMap, ok := payloadVal.(map[string]any)
	if !ok {
		return nil, nil
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshal payload: %w", err)
	}

	return a.processEntry(entryType, timestamp, payloadBytes)
}

// AdaptBatch converts multiple raw JSONL lines to v2 Session records.
// Uses two-pass processing on pre-parsed entries: all lines are parsed once into
// (entryType, timestamp, payloadBytes) tuples, then metadata entries are processed
// first to populate cached state, followed by content entry conversion.
// This avoids O(2n) JSON parsing that would occur from parsing each line twice.
//
// Invalid lines are skipped with a warning log rather than failing the entire batch,
// matching the per-event resilience pattern used by Claude and Gemini handlers.
func (a *Adapter) AdaptBatch(lines []string) []*session.Session {
	// Step 1 - Parse all lines once
	var entries []*parsedEntry
	for i, line := range lines {
		entry, err := parseEntry(line)
		if err != nil {
			a.logger.Warn("failed to parse JSONL line in batch",
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}
		if entry == nil {
			continue
		}
		entries = append(entries, entry)
	}

	// Step 2 - Process metadata entries first
	for _, entry := range entries {
		if !metadataTypes[entry.entryType] {
			continue
		}
		if err := a.processMetadata(entry.entryType, entry.payloadBytes); err != nil {
			a.logger.Warn("failed to process metadata entry",
				slog.String("type", entry.entryType),
				slog.Any("error", err),
			)
		}
	}

	// Step 3 - Convert content entries
	var sessions []*session.Session
	for _, entry := range entries {
		if metadataTypes[entry.entryType] {
			continue
		}
		converted, err := a.processContent(entry.entryType, entry.timestamp, entry.payloadBytes)
		if err != nil {
			a.logger.Warn("failed to convert content entry",
				slog.String("type", entry.entryType),
				slog.Any("error", err),
			)
			continue
		}
		sessions = append(sessions, converted...)
	}

	return sessions
}

// parseEntry parses a raw JSONL line into a parsedEntry.
// Returns nil for empty lines or lines without the required fields.
func parseEntry(line string) (*parsedEntry, error) {
	if line == "" {
		return nil, nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSONL line: %w", err)
	}

	typeVal, ok := raw["type"]
	if !ok {
		return nil, nil
	}
	entryType, ok := typeVal.(string)
	if !ok {
		return nil, nil
	}

	timestamp := extractTimestamp(raw)

	payloadVal, ok := raw["payload"]
	if !ok {
		return nil, nil
	}
	payloadMap, ok := payloadVal.(map[string]any)
	if !ok {
		return nil, nil
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshal payload: %w", err)
	}

	return &parsedEntry{
		entryType:    entryType,
		timestamp:    timestamp,
		payloadBytes: payloadBytes,
	}, nil
}

// processEntry dispatches a parsed entry to the appropriate handler.
func (a *Adapter) processEntry(entryType string, timestamp time.Time, payloadBytes []byte) ([]*session.Session, error) {
	switch entryType {
	case "session_meta":
		var payload SessionMetaPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session_meta: %w", err)
		}
		a.cached.sessionID = payload.ID
		if payload.ModelProvider != nil {
			a.cached.modelProvider = *payload.ModelProvider
		}
		return nil, nil

	case "turn_context":
		var payload TurnContextPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal turn_context: %w", err)
		}
		a.cached.model = payload.Model
		return nil, nil

	case "event_msg":
		var payload EventMsgPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event_msg: %w", err)
		}
		return a.adaptEventMsg(&payload, timestamp), nil

	case "response_item":
		var payload ResponseItemPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response_item: %w", err)
		}
		return a.adaptResponseItem(&payload, timestamp), nil

	default:
		return nil, nil
	}
}

// processMetadata handles metadata-only entries (session_meta, turn_context).
func (a *Adapter) processMetadata(entryType string, payloadBytes []byte) error {
	switch entryType {
	case "session_meta":
		var payload SessionMetaPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal session_meta: %w", err)
		}
		a.cached.sessionID = payload.ID
		if payload.ModelProvider != nil {
			a.cached.modelProvider = *payload.ModelProvider
		}
		return nil

	case "turn_context":
		var payload TurnContextPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal turn_context: %w", err)
		}
		a.cached.model = payload.Model
		return nil

	default:
		return nil
	}
}

// processContent handles content entries (event_msg, response_item).
func (a *Adapter) processContent(entryType string, timestamp time.Time, payloadBytes []byte) ([]*session.Session, error) {
	switch entryType {
	case "event_msg":
		var payload EventMsgPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event_msg: %w", err)
		}
		return a.adaptEventMsg(&payload, timestamp), nil

	case "response_item":
		var payload ResponseItemPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response_item: %w", err)
		}
		return a.adaptResponseItem(&payload, timestamp), nil

	default:
		return nil, nil
	}
}

// buildTreeNodeMeta creates a TreeNodeMeta with the adapter's cached state.
// Centralizes UUID generation, Provider setting, and SessionID population.
func (a *Adapter) buildTreeNodeMeta(timestamp time.Time) session.TreeNodeMeta {
	return session.TreeNodeMeta{
		UUID:      newUUID(),
		SessionID: a.cached.sessionID,
		Timestamp: timestamp,
		Provider:  ProviderName,
	}
}

// extractTimestamp parses the "timestamp" field from a raw JSON map.
// Returns time.Now() as fallback if the field is missing or unparseable.
func extractTimestamp(raw map[string]any) time.Time {
	tsVal, ok := raw["timestamp"]
	if !ok {
		return time.Now()
	}
	tsStr, ok := tsVal.(string)
	if !ok {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return time.Now()
	}
	return t
}

// newUUID generates a UUID v4 string using crypto/rand.
// Uses stdlib to avoid adding github.com/google/uuid as a dependency to the shared module.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
