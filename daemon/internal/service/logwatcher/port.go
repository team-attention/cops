package logwatcher

// FileOp represents a file operation type.
type FileOp uint32

const (
	// OpCreate indicates a file was created.
	OpCreate FileOp = 1 << iota
	// OpWrite indicates a file was written.
	OpWrite
	// OpRemove indicates a file was removed.
	OpRemove
	// OpRename indicates a file was renamed.
	OpRename
	// OpChmod indicates a file's permissions changed.
	OpChmod
)

// FileEvent represents a file system event.
type FileEvent struct {
	Path string
	Op   FileOp
}

// Has checks if the event has the specified operation.
func (e FileEvent) Has(op FileOp) bool {
	return e.Op&op != 0
}

// FileWatchPort is the port interface for file watching.
type FileWatchPort interface {
	// Add adds a path to watch.
	Add(path string) error
	// Remove removes a path from watching.
	Remove(path string) error
	// Events returns the channel for file events.
	Events() <-chan FileEvent
	// Errors returns the channel for errors.
	Errors() <-chan error
	// Close stops the watcher and releases resources.
	Close() error
}
