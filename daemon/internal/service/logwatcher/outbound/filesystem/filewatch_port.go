package filesystem

// FileWatchPort defines the interface for file watching operations.
type FileWatchPort interface {
	// Add adds a directory to watch.
	Add(path string) error
	// Remove removes a directory from watching.
	Remove(path string) error
}
