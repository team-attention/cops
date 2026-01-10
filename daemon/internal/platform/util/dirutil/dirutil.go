package dirutil

import (
	"os"
	"path/filepath"
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
