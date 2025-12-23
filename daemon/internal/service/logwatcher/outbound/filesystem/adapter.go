package filesystem

import (
	"github.com/fsnotify/fsnotify"

	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
)

// Adapter implements FileWatchPort using fsnotify.
type Adapter struct {
	watcher *fsnotify.Watcher
	events  chan logwatcher.FileEvent
	errors  chan error
	done    chan struct{}
}

// NewAdapter creates a new filesystem watch adapter.
func NewAdapter() (*Adapter, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	a := &Adapter{
		watcher: watcher,
		events:  make(chan logwatcher.FileEvent),
		errors:  make(chan error),
		done:    make(chan struct{}),
	}

	go a.loop()
	return a, nil
}

// Add adds a path to watch.
func (a *Adapter) Add(path string) error {
	return a.watcher.Add(path)
}

// Remove removes a path from watching.
func (a *Adapter) Remove(path string) error {
	return a.watcher.Remove(path)
}

// Events returns the channel for file events.
func (a *Adapter) Events() <-chan logwatcher.FileEvent {
	return a.events
}

// Errors returns the channel for errors.
func (a *Adapter) Errors() <-chan error {
	return a.errors
}

// Close stops the watcher and releases resources.
func (a *Adapter) Close() error {
	close(a.done)
	return a.watcher.Close()
}

func (a *Adapter) loop() {
	for {
		select {
		case <-a.done:
			return
		case event, ok := <-a.watcher.Events:
			if !ok {
				return
			}
			a.events <- logwatcher.FileEvent{
				Path: event.Name,
				Op:   convertOp(event.Op),
			}
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			a.errors <- err
		}
	}
}

func convertOp(op fsnotify.Op) logwatcher.FileOp {
	var result logwatcher.FileOp
	if op.Has(fsnotify.Create) {
		result |= logwatcher.OpCreate
	}
	if op.Has(fsnotify.Write) {
		result |= logwatcher.OpWrite
	}
	if op.Has(fsnotify.Remove) {
		result |= logwatcher.OpRemove
	}
	if op.Has(fsnotify.Rename) {
		result |= logwatcher.OpRename
	}
	if op.Has(fsnotify.Chmod) {
		result |= logwatcher.OpChmod
	}
	return result
}
