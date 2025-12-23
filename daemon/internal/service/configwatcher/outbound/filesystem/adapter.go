package filesystem

import (
	"github.com/fsnotify/fsnotify"

	"github.com/team-attention/cops/daemon/internal/service/configwatcher"
)

// Adapter implements FileWatchPort using fsnotify.
type Adapter struct {
	watcher *fsnotify.Watcher
	events  chan configwatcher.FileEvent
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
		events:  make(chan configwatcher.FileEvent),
		errors:  make(chan error),
		done:    make(chan struct{}),
	}

	go a.loop()
	return a, nil
}

// Watch starts watching a file path.
func (a *Adapter) Watch(path string) error {
	return a.watcher.Add(path)
}

// Events returns the channel for file events.
func (a *Adapter) Events() <-chan configwatcher.FileEvent {
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
			a.events <- configwatcher.FileEvent{
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

func convertOp(op fsnotify.Op) configwatcher.FileOp {
	var result configwatcher.FileOp
	if op.Has(fsnotify.Create) {
		result |= configwatcher.OpCreate
	}
	if op.Has(fsnotify.Write) {
		result |= configwatcher.OpWrite
	}
	if op.Has(fsnotify.Remove) {
		result |= configwatcher.OpRemove
	}
	if op.Has(fsnotify.Rename) {
		result |= configwatcher.OpRename
	}
	if op.Has(fsnotify.Chmod) {
		result |= configwatcher.OpChmod
	}
	return result
}
