package domain

// Provider represents the source coding tool that produces log files.
type Provider string

const (
	// ProviderClaudeCode is the Claude Code CLI.
	ProviderClaudeCode Provider = "claude"
	// ProviderGeminiCLI is the Google Gemini CLI.
	ProviderGeminiCLI Provider = "gemini_cli"
	// ProviderCodexCLI is the OpenAI Codex CLI.
	ProviderCodexCLI Provider = "codex_cli"
	// ProviderOpenCode is the OpenCode CLI.
	ProviderOpenCode Provider = "opencode"
)

// AllProviders returns all supported providers.
func AllProviders() []Provider {
	return []Provider{
		ProviderClaudeCode,
		ProviderGeminiCLI,
		ProviderCodexCLI,
		ProviderOpenCode,
	}
}
