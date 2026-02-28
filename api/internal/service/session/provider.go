package session

// Provider represents the AI provider type.
type Provider string

const (
	// ProviderClaudeCode represents Claude Code logs (JSONL format).
	ProviderClaudeCode Provider = "claude_code"
	// ProviderGeminiCLI represents Gemini CLI logs (JSON session format).
	ProviderGeminiCLI Provider = "gemini_cli"
	// ProviderOpenCode represents OpenCode logs (structured JSON from SQLite via daemon polling).
	ProviderOpenCode Provider = "open_code"
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

// DetectProvider analyzes event data to determine which provider it came from.
// The data parameter is expected to be a map[string]any from parsed JSON.
func DetectProvider(data any) Provider {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return ProviderUnknown
	}

	if isClaudeCodeFormat(dataMap) {
		return ProviderClaudeCode
	}

	if isOpenCodeFormat(dataMap) {
		return ProviderOpenCode
	}

	if isGeminiCLIFormat(dataMap) {
		return ProviderGeminiCLI
	}

	return ProviderUnknown
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

// isOpenCodeFormat checks if the JSON structure matches OpenCode message format.
//
// Detection strategy:
// OpenCode messages are identified by the combination of three fields:
//   - "sessionId" (camelCase string) - Same key name as Gemini CLI, but OpenCode
//     lacks the "messages" array that Gemini has. Since Gemini is checked after
//     OpenCode, and Gemini requires both "sessionId" + "messages" array, there
//     is no collision.
//   - "role" (string: "user" or "assistant") - Confirms this is a message record.
//     Claude Code also has a "type" field with similar values but uses different
//     key name, and Claude Code is already matched first in DetectProvider.
//   - "parts" (string, not array) - The parts field is stored as TEXT in SQLite
//     and serialized as a JSON string by the daemon. This is unique to OpenCode;
//     other providers that have parts/content use arrays or objects.
//
// Non-collision guarantees:
//   - Claude Code: Detected first via "type" field (isClaudeCodeFormat). Even if
//     both formats had overlapping fields, Claude Code takes priority.
//   - Gemini CLI: Uses "sessionId" (same key) but also requires "messages" (array).
//     OpenCode has "parts" (string) instead. The two are mutually exclusive.
//   - Unknown formats: Require all three conditions (sessionId + role + string parts)
//     making false positives extremely unlikely.
func isOpenCodeFormat(data map[string]any) bool {
	if _, ok := data["sessionId"]; !ok {
		return false
	}

	roleVal, ok := data["role"]
	if !ok {
		return false
	}

	roleStr, ok := roleVal.(string)
	if !ok {
		return false
	}

	if roleStr != "user" && roleStr != "assistant" {
		return false
	}

	partsVal, ok := data["parts"]
	if !ok {
		return false
	}

	_, ok = partsVal.(string)
	return ok
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
