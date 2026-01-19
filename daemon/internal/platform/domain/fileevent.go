package domain

// FileOp represents file operation types from fsnotify.
type FileOp uint32

const (
	// OpCreate represents file creation event.
	OpCreate FileOp = 1 << iota
	// OpWrite represents file write event.
	OpWrite
	// OpRemove represents file removal event.
	OpRemove
	// OpRename represents file rename event.
	OpRename
	// OpChmod represents file permission change event.
	OpChmod
)

// FileEvent represents a file system event.
type FileEvent struct {
	Path string // File path where the event occurred
	Op   FileOp // Type of file operation
}
