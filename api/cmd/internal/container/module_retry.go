package container

import (
	"context"

	"go.uber.org/fx"

	retryservice "github.com/team-attention/cops/api/internal/service/retry"
	"github.com/team-attention/cops/api/internal/service/retry/inbound/worker/ticker"
	"github.com/team-attention/cops/api/internal/service/retry/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/retry/outbound/repository/mongodb"
)

// newRetryModule creates the fx module for retry service components.
func newRetryModule() fx.Option {
	return fx.Module("retry",
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoRetryRepository,
				fx.As(new(repository.RetryRepositoryPort)),
			),
		),

		fx.Provide(retryservice.NewService),

		fx.Provide(ticker.NewRetryWorkerHandler),

		fx.Invoke(registerRetryWorker),
	)
}

// registerRetryWorker registers the retry worker with fx lifecycle.
// This follows the standard fx.Invoke pattern used throughout the codebase.
func registerRetryWorker(lc fx.Lifecycle, handler *ticker.RetryWorkerHandler) {
	if !handler.IsEnabled() {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			handler.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			handler.Stop()
			return nil
		},
	})
}
