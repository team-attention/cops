package pathutil

import (
	"os"
	"path/filepath"
)

// GetGeminiLogBaseDir returns the Gemini CLI log base directory.
// Returns "~/.gemini/tmp" expanded to absolute path.
func GetGeminiLogBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gemini", "tmp")
}

// GetCodexSessionsBaseDir returns the Codex CLI sessions base directory.
// Returns "~/.codex/sessions" expanded to absolute path.
func GetCodexSessionsBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "sessions")
}

// GetOpenCodeDataDir returns the OpenCode data directory.
// Returns "~/.local/share/opencode" expanded to absolute path.
func GetOpenCodeDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode")
}

// GetOpenCodeDBPath returns the OpenCode SQLite database path.
// Returns "~/.local/share/opencode/opencode.db" expanded to absolute path.
func GetOpenCodeDBPath() string {
	dataDir := GetOpenCodeDataDir()
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, "opencode.db")
}
