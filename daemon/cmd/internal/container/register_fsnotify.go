package container

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
)

// FsnotifyHandler is implemented by Inbound handlers that process fsnotify events.
type FsnotifyHandler interface {
	// Start begins the event loop (non-blocking).
	Start(ctx context.Context) error
	// Stop gracefully shuts down the event loop.
	Stop(ctx context.Context) error
}

type fsnotifyParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *slog.Logger
	Handlers  []FsnotifyHandler `group:"fsnotify_handlers"`
}

// registerFsnotify collects all FsnotifyHandler implementations and manages their lifecycle.
func registerFsnotify(p fsnotifyParams) {
	p.Logger.Info("registering fsnotify handlers",
		slog.Int("count", len(p.Handlers)),
	)

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			for _, handler := range p.Handlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("all fsnotify handlers started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop handlers in reverse order
			for i := len(p.Handlers) - 1; i >= 0; i-- {
				handler := p.Handlers[i]
				_ = handler.Stop(ctx)
				// Continue stopping other handlers even if one fails
			}
			p.Logger.Info("all fsnotify handlers stopped")
			return nil
		},
	})
}
