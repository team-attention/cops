package pubsub

// WriterPort defines publish-only interface for pub/sub pattern.
type WriterPort[T any] interface {
	// Publish sends a message to all subscribers.
	Publish(msg T) error
}

// ReaderPort defines subscribe-only interface for pub/sub pattern.
type ReaderPort[T any] interface {
	// Subscribe returns a channel to receive messages.
	// The caller must call Unsubscribe when done to prevent goroutine leaks.
	Subscribe() <-chan T
	// Unsubscribe removes the subscriber and closes the channel.
	Unsubscribe(ch <-chan T)
}
