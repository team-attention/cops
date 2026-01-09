package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository/mongodb"
)

func newEventModule() fx.Option {
	return fx.Module("event",
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoEventRepository,
				fx.As(new(repository.EventRepositoryPort)),
			),
		),
	)
}
