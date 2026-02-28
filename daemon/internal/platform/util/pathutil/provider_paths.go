package pathutil

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const (
	// geminiHashLength is the number of hex characters used for Gemini project hashes.
	// Gemini CLI truncates the SHA-256 hex digest to this length.
	// Verified against gemini-cli v0.2.x source: src/utils/hash.ts.
	geminiHashLength = 40
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

// GeminiProjectHash computes the Gemini CLI project hash for a given absolute path.
// Gemini CLI uses a truncated hex-encoded SHA-256 hash of the absolute project path.
func GeminiProjectHash(projectPath string) string {
	h := sha256.Sum256([]byte(projectPath))
	return hex.EncodeToString(h[:])[:geminiHashLength]
}

// BuildGeminiHashToPathMap creates a mapping from Gemini project hash to project path
// for a given set of registered project paths.
func BuildGeminiHashToPathMap(projectPaths []string) map[string]string {
	m := make(map[string]string, len(projectPaths))
	for _, p := range projectPaths {
		m[GeminiProjectHash(p)] = p
	}
	return m
}

// ExtractGeminiProjectHash extracts the project hash from a Gemini log directory path.
// Given a path like "~/.gemini/tmp/{hash}/chats", returns the {hash} component.
// Returns empty string if the path does not match the expected Gemini directory structure.
func ExtractGeminiProjectHash(logDir string) string {
	baseDir := GetGeminiLogBaseDir()
	if baseDir == "" {
		return ""
	}

	prefix := baseDir + string(filepath.Separator)
	if !strings.HasPrefix(logDir, prefix) {
		return ""
	}

	remaining := strings.TrimPrefix(logDir, prefix)
	parts := strings.SplitN(remaining, string(filepath.Separator), 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}

	return parts[0]
}
