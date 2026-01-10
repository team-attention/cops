package mock

import (
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
)

// FileWatch implements filesystem.FileWatchPort for testing.
type FileWatch struct {
	AddFunc       func(path string) error
	RemoveFunc    func(path string) error
	WatchListFunc func() []string
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

// WatchList implements filesystem.FileWatchPort.
func (m *FileWatch) WatchList() []string {
	// 1. If WatchListFunc is set, call it and return result
	if m.WatchListFunc != nil {
		return m.WatchListFunc()
	}

	// 2. Otherwise return empty slice
	return []string{}
}

// Compile-time interface verification.
var _ filesystem.FileWatchPort = (*FileWatch)(nil)
