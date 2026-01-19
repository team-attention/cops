package fsnotify

import (
	"github.com/fsnotify/fsnotify"
)

// FileWatchAdapter wraps *fsnotify.Watcher to implement FileWatchPort.
// This adapter only handles Add/Remove operations.
// The same watcher is also used by Inbound handler to read Events.
type FileWatchAdapter struct {
	watcher *fsnotify.Watcher
}

// NewFileWatchAdapter creates a new FileWatchAdapter.
func NewFileWatchAdapter(watcher *fsnotify.Watcher) *FileWatchAdapter {
	return &FileWatchAdapter{
		watcher: watcher,
	}
}

// Add adds a directory to watch.
func (a *FileWatchAdapter) Add(path string) error {
	return a.watcher.Add(path)
}

// Remove removes a directory from watching.
func (a *FileWatchAdapter) Remove(path string) error {
	return a.watcher.Remove(path)
}

// WatchList returns list of currently watched paths.
func (a *FileWatchAdapter) WatchList() []string {
	return a.watcher.WatchList()
}
