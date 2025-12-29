package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// EncodePathForClaude converts a file path to Claude Code's encoded format.
// e.g., "/Users/jayce/project" → "-Users-jayce-project"
func EncodePathForClaude(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}

// GetClaudeProjectDir returns the Claude Code project directory for a given path.
// e.g., "/Users/jayce/project" → "~/.claude/projects/-Users-jayce-project"
func GetClaudeProjectDir(projectPath string) string {
	encoded := EncodePathForClaude(projectPath)
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects", encoded)
}

// GetClaudeProjectsBaseDir returns the base directory for Claude Code projects.
// e.g., "~/.claude/projects"
func GetClaudeProjectsBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

// DecodeClaudeProjectDir decodes a Claude Code project directory path back to the original file system path.
// e.g., "~/.claude/projects/-Users-jayce-project" → "/Users/jayce/project"
// Returns empty string if the path is not a valid Claude project directory.
func DecodeClaudeProjectDir(claudeDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Build expected prefix
	prefix := filepath.Join(home, ".claude", "projects") + string(filepath.Separator)

	// Check if claudeDir starts with prefix
	if !strings.HasPrefix(claudeDir, prefix) {
		return ""
	}

	// Extract encoded portion
	encoded := strings.TrimPrefix(claudeDir, prefix)
	if encoded == "" {
		return ""
	}

	// Decode: replace all '-' with '/'
	decoded := strings.ReplaceAll(encoded, "-", "/")

	// Ensure it starts with '/'
	if !strings.HasPrefix(decoded, "/") {
		return ""
	}

	return decoded
}
