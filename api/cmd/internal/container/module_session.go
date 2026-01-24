package container

import (
	"context"

	"go.uber.org/fx"

	sessionservice "github.com/team-attention/cops/api/internal/service/session"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository/mongodb"
)

// newSessionModule creates the fx module for Session Service.
// Note: Adapters are created inline in Service constructor since they have no dependencies.
func newSessionModule() fx.Option {
	return fx.Module("session",
		// Repositories
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoSessionRepository,
				fx.As(new(repository.SessionRepositoryPort)),
			),
		),
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoResumeTokenRepository,
				fx.As(new(repository.ResumeTokenRepositoryPort)),
			),
		),
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoEventQuery,
				fx.As(new(repository.EventQueryPort)),
			),
		),

		// Service (adapters created inline, not injected)
		fx.Provide(sessionservice.NewService),

		// Lifecycle registration
		fx.Invoke(registerSessionLifecycle),
	)
}

// sessionLifecycleParams collects dependencies for lifecycle registration.
type sessionLifecycleParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Service   *sessionservice.Service
}

// registerSessionLifecycle registers fx lifecycle hooks for Session Service.
func registerSessionLifecycle(params sessionLifecycleParams) {
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return params.Service.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return params.Service.Stop(ctx)
		},
	})
}
