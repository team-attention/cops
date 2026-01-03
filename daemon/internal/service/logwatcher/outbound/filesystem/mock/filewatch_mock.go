package mock

import (
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
)

// FileWatch implements filesystem.FileWatchPort for testing.
type FileWatch struct {
	AddFunc    func(path string) error
	RemoveFunc func(path string) error
}

// Add implements filesystem.FileWatchPort.
func (m *FileWatch) Add(path string) error {
	// 1. If AddFunc is set, call it and return result
	if m.AddFunc != nil {
		return m.AddFunc(path)
	}

	// 2. Otherwise return nil
	return nil
}

// Remove implements filesystem.FileWatchPort.
func (m *FileWatch) Remove(path string) error {
	// 1. If RemoveFunc is set, call it and return result
	if m.RemoveFunc != nil {
		return m.RemoveFunc(path)
	}

	// 2. Otherwise return nil
	return nil
}

// Compile-time interface verification.
var _ filesystem.FileWatchPort = (*FileWatch)(nil)
