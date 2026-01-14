package dirutil

import (
	"os"
	"path/filepath"
	"strings"
)

// WalkDirectories returns all directories (including nested) under the given root.
// Excludes hidden directories (starting with '.').
func WalkDirectories(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}

		if d.IsDir() {
			// Skip hidden directories
			if len(d.Name()) > 0 && d.Name()[0] == '.' {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return dirs, nil
}

// FindJSONLFiles returns all .jsonl files in the given directory.
func FindJSONLFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}
