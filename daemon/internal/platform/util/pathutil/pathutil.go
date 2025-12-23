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
