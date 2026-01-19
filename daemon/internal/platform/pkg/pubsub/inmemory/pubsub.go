package inmemory

import (
	"log/slog"
	"sync"
)

// PubSub is a generic in-memory pub/sub implementation.
// It is thread-safe and uses buffered channels for subscribers.
type PubSub[T any] struct {
	logger      *slog.Logger
	subscribers map[<-chan T]chan T
	mu          sync.RWMutex
	bufferSize  int
}

// NewPubSub creates a new in-memory pub/sub instance.
// bufferSize determines the channel buffer size for each subscriber.
func NewPubSub[T any](l *slog.Logger, bufferSize int) *PubSub[T] {
	if bufferSize <= 0 {
		bufferSize = 10 // default buffer size
	}
	return &PubSub[T]{
		logger:      l.With(slog.String("name", "pubsub.inmemory")),
		subscribers: make(map[<-chan T]chan T),
		bufferSize:  bufferSize,
	}
}

// Publish sends a message to all subscribers.
// If a subscriber's channel is full, the message is dropped for that subscriber.
func (ps *PubSub[T]) Publish(msg T) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	ps.logger.Debug("publishing message",
		slog.Int("subscribers", len(ps.subscribers)),
	)

	for _, ch := range ps.subscribers {
		select {
		case ch <- msg:
			// Message sent successfully
		default:
			// Channel is full, drop message for this subscriber
			ps.logger.Warn("subscriber channel full, message dropped")
		}
	}
	return nil
}

// Subscribe returns a read-only channel to receive messages.
// The caller must call Unsubscribe when done to prevent goroutine leaks.
func (ps *PubSub[T]) Subscribe() <-chan T {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan T, ps.bufferSize)
	ps.subscribers[ch] = ch

	ps.logger.Debug("new subscriber added",
		slog.Int("total_subscribers", len(ps.subscribers)),
	)

	return ch
}

// Unsubscribe removes the subscriber and closes the channel.
func (ps *PubSub[T]) Unsubscribe(ch <-chan T) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if writeCh, ok := ps.subscribers[ch]; ok {
		close(writeCh)
		delete(ps.subscribers, ch)

		ps.logger.Debug("subscriber removed",
			slog.Int("remaining_subscribers", len(ps.subscribers)),
		)
	}
}

// Close closes all subscriber channels and clears the subscriber list.
func (ps *PubSub[T]) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for ch, writeCh := range ps.subscribers {
		close(writeCh)
		delete(ps.subscribers, ch)
	}

	ps.logger.Info("pubsub closed, all subscribers removed")
}
