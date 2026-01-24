package session

// Provider represents the AI provider type.
type Provider string

const (
	// ProviderClaudeCode represents Claude Code logs (JSONL format).
	ProviderClaudeCode Provider = "claude_code"
	// ProviderGeminiCLI represents Gemini CLI logs (JSON session format).
	ProviderGeminiCLI Provider = "gemini_cli"
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
