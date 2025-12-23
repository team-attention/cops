package gitutil

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GetWorktrees returns all worktree paths for a Git repository.
// The main repository path is included as the first element.
func GetWorktrees(repoPath string) ([]string, error) {
	worktrees := []string{repoPath}

	worktreesDir := filepath.Join(repoPath, ".git", "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return worktrees, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		gitdirPath := filepath.Join(worktreesDir, entry.Name(), "gitdir")
		worktreePath, err := readWorktreePath(gitdirPath)
		if err != nil {
			continue
		}

		worktrees = append(worktrees, worktreePath)
	}

	return worktrees, nil
}

// readWorktreePath reads the worktree path from a gitdir file.
func readWorktreePath(gitdirPath string) (string, error) {
	file, err := os.Open(gitdirPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// The gitdir file contains the path to the .git file in the worktree
		// e.g., "/Users/jayce/project/.worktrees/feature-branch/.git"
		// We need to get the parent directory
		return filepath.Dir(line), nil
	}

	return "", scanner.Err()
}

// IsGitRepository checks if a path is a Git repository.
func IsGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// .git can be a directory (normal repo) or a file (worktree)
	return info.IsDir() || info.Mode().IsRegular()
}
