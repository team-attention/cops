package container

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
)

// SubscriberHandler is implemented by handlers that subscribe to pubsub.
// These handlers MUST be started before PublisherHandlers.
type SubscriberHandler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// PublisherHandler is implemented by handlers that publish to pubsub.
// These handlers MUST be started after SubscriberHandlers.
type PublisherHandler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// FsnotifyHandler is implemented by handlers that only watch files (no pubsub).
type FsnotifyHandler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type handlerParams struct {
	fx.In

	Lifecycle          fx.Lifecycle
	Logger             *slog.Logger
	SubscriberHandlers []SubscriberHandler `group:"subscriber_handlers"`
	PublisherHandlers  []PublisherHandler  `group:"publisher_handlers"`
	FsnotifyHandlers   []FsnotifyHandler   `group:"fsnotify_handlers"`
}

// registerHandlers manages the lifecycle of all handlers with correct start order.
// Order: SubscriberHandlers -> PublisherHandlers -> FsnotifyHandlers
func registerHandlers(p handlerParams) {
	totalHandlers := len(p.SubscriberHandlers) + len(p.PublisherHandlers) + len(p.FsnotifyHandlers)
	p.Logger.Info("registering handlers",
		slog.Int("subscriber_handlers", len(p.SubscriberHandlers)),
		slog.Int("publisher_handlers", len(p.PublisherHandlers)),
		slog.Int("fsnotify_handlers", len(p.FsnotifyHandlers)),
		slog.Int("total", totalHandlers),
	)

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 1. Start subscriber handlers first (they need to subscribe before publishers publish)
			for _, handler := range p.SubscriberHandlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("subscriber handlers started", slog.Int("count", len(p.SubscriberHandlers)))

			// 2. Start publisher handlers (they can now publish to subscribers)
			for _, handler := range p.PublisherHandlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("publisher handlers started", slog.Int("count", len(p.PublisherHandlers)))

			// 3. Start fsnotify handlers (no pubsub dependency)
			for _, handler := range p.FsnotifyHandlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("fsnotify handlers started", slog.Int("count", len(p.FsnotifyHandlers)))

			p.Logger.Info("all handlers started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop in reverse order: fsnotify -> publishers -> subscribers
			for i := len(p.FsnotifyHandlers) - 1; i >= 0; i-- {
				_ = p.FsnotifyHandlers[i].Stop(ctx)
			}
			for i := len(p.PublisherHandlers) - 1; i >= 0; i-- {
				_ = p.PublisherHandlers[i].Stop(ctx)
			}
			for i := len(p.SubscriberHandlers) - 1; i >= 0; i-- {
				_ = p.SubscriberHandlers[i].Stop(ctx)
			}
			p.Logger.Info("all handlers stopped")
			return nil
		},
	})
}
