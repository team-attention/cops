package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to home directory and resolves to absolute path.
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}

// EncodePathForClaude encodes a path for Claude's project directory naming.
// e.g., /Users/jayce/project -> -Users-jayce-project
func EncodePathForClaude(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "-")
}

// GetClaudeProjectDir returns the Claude projects directory for a given path.
func GetClaudeProjectDir(claudeBaseDir, projectPath string) string {
	encoded := EncodePathForClaude(projectPath)
	return filepath.Join(claudeBaseDir, "projects", encoded)
}

// DefaultClaudeDir returns the default Claude configuration directory (~/.claude).
func DefaultClaudeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// DefaultCopsConfigDir returns the default cops configuration directory (~/.cops).
func DefaultCopsConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cops"), nil
}

// EnsureDir ensures a directory exists, creating it if necessary.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
