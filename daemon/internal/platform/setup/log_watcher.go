package setup

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// InitLogWatcher creates a shared fsnotify.Watcher for log file watching.
// This watcher is shared between Inbound (reads Events) and Outbound (calls Add/Remove).
func InitLogWatcher(l *slog.Logger) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	l.Info("log watcher initialized")
	return watcher, nil
}
