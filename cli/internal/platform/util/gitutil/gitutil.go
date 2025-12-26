package gitutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotGitRepo is returned when a directory is not a git repository.
var ErrNotGitRepo = errors.New("not a git repository")

// IsGitRepo checks if a directory is a git repository.
func IsGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	// .git can be either a directory (regular repo) or a file (worktree)
	return info.IsDir() || info.Mode().IsRegular()
}

// FindMainRepoPath finds the main repository path from a directory.
// If the directory is a worktree, it returns the main repo path.
// If it's the main repo, it returns the directory itself.
func FindMainRepoPath(dir string) (string, error) {
	if !IsGitRepo(dir) {
		return "", ErrNotGitRepo
	}

	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", err
	}

	// If .git is a directory, this is the main repo
	if info.IsDir() {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		return absPath, nil
	}

	// If .git is a file, this is a worktree
	// Read the .git file to find the main repo
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}

	// Format: "gitdir: /path/to/main/.git/worktrees/worktree-name"
	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", errors.New("invalid .git file format")
	}

	gitDirPath := strings.TrimPrefix(line, "gitdir: ")

	// Navigate up from .git/worktrees/name to find the main repo
	// gitDirPath is like /path/to/main/.git/worktrees/worktree-name
	// We need to go up 3 levels to get /path/to/main
	mainGitDir := filepath.Dir(filepath.Dir(filepath.Dir(gitDirPath)))
	mainRepoPath := filepath.Dir(mainGitDir)

	return mainRepoPath, nil
}

// ListWorktrees returns all worktree paths for a git repository.
func ListWorktrees(mainRepoPath string) ([]string, error) {
	worktreesDir := filepath.Join(mainRepoPath, ".git", "worktrees")

	// Check if worktrees directory exists
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		return nil, nil // No worktrees
	}

	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return nil, err
	}

	var worktrees []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		gitdirFile := filepath.Join(worktreesDir, entry.Name(), "gitdir")
		content, err := os.ReadFile(gitdirFile)
		if err != nil {
			continue
		}

		// The gitdir file contains the path to the worktree's .git file
		// We need to get the parent directory
		worktreePath := filepath.Dir(strings.TrimSpace(string(content)))
		worktrees = append(worktrees, worktreePath)
	}

	return worktrees, nil
}

// GetCurrentBranch returns the current branch name for a git repository.
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the configured remote URL for a git repository.
// Uses: git remote get-url origin
func GetRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetActualRemoteURL returns the actual remote URL (what GitHub points to).
// Uses: git ls-remote --get-url origin
// This can differ from configured URL if the repo was renamed on GitHub.
// Returns empty string on error (graceful handling).
func GetActualRemoteURL(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "ls-remote", "--get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

