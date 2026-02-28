package session

// Provider represents the AI provider type.
type Provider string

const (
	// ProviderClaudeCode represents Claude Code logs (JSONL format).
	ProviderClaudeCode Provider = "claude_code"
	// ProviderGeminiCLI represents Gemini CLI logs (JSON session format).
	ProviderGeminiCLI Provider = "gemini_cli"
	// ProviderCodexCLI represents Codex CLI logs (JSONL format).
	ProviderCodexCLI Provider = "codex_cli"
	// ProviderUnknown represents unrecognized format.
	ProviderUnknown Provider = "unknown"
)

// claudeCodeTypes are the valid top-level type values for Claude Code transcripts.
var claudeCodeTypes = map[string]bool{
	"user":                  true,
	"assistant":             true,
	"system":                true,
	"summary":               true,
	"file-history-snapshot": true,
	"progress":              true,
}

// codexCLITypes are the valid top-level type values for Codex CLI entries.
// Includes both metadata types (session_meta, turn_context) used by the adapter for
// state caching, and content types (event_msg, response_item) that produce v2 sessions.
// This differs from the adapter's internal metadataTypes map, which only contains the
// metadata subset for two-pass batch processing.
var codexCLITypes = map[string]bool{
	"session_meta":  true,
	"event_msg":     true,
	"response_item": true,
	"turn_context":  true,
}

// DetectProvider analyzes event data to determine which provider it came from.
// The data parameter is expected to be a map[string]any from parsed JSON.
//
// Detection order:
// 1. Codex CLI (has "type" + "payload") - checked first because both Codex and Claude
//    use "type", but only Codex has "payload". Checking Codex first prevents false
//    positives if Codex type values were to overlap with Claude type values.
// 2. Claude Code (has "type" with Claude-specific values, no "payload")
// 3. Gemini CLI (has "sessionId" + "messages")
func DetectProvider(data any) Provider {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return ProviderUnknown
	}

	if isCodexCLIFormat(dataMap) {
		return ProviderCodexCLI
	}

	if isClaudeCodeFormat(dataMap) {
		return ProviderClaudeCode
	}

	if isGeminiCLIFormat(dataMap) {
		return ProviderGeminiCLI
	}

	return ProviderUnknown
}

// isCodexCLIFormat checks if the JSON structure matches Codex CLI format.
// Codex entries have both a "type" field (with Codex-specific values) and a "payload" field.
// The "payload" field is the decisive discriminator: Claude Code entries also have a "type"
// field but never contain a "payload" field. This allows unambiguous detection even when
// Codex type values overlap with Claude Code type values (they currently do not, but the
// "payload" check provides a more robust structural guarantee).
func isCodexCLIFormat(data map[string]any) bool {
	typeVal, ok := data["type"]
	if !ok {
		return false
	}
	typeStr, ok := typeVal.(string)
	if !ok {
		return false
	}
	if !codexCLITypes[typeStr] {
		return false
	}
	if _, ok := data["payload"]; !ok {
		return false
	}
	return true
}

// isClaudeCodeFormat checks if the JSON structure matches Claude Code format.
func isClaudeCodeFormat(data map[string]any) bool {
	typeVal, ok := data["type"]
	if !ok {
		return false
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		return false
	}

	return claudeCodeTypes[typeStr]
}

// isGeminiCLIFormat checks if the JSON structure matches Gemini CLI format.
func isGeminiCLIFormat(data map[string]any) bool {
	if _, ok := data["sessionId"]; !ok {
		return false
	}

	messagesVal, ok := data["messages"]
	if !ok {
		return false
	}

	_, ok = messagesVal.([]any)
	return ok
}
