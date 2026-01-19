package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/invitation"
	"github.com/team-attention/cops/api/internal/service/invitation/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/invitation/outbound/repository/mongodb"
)

func newInvitationModule() fx.Option {
	return fx.Module("invitation",
		// Invitation repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoInvitationRepository,
				fx.As(new(repository.InvitationRepositoryPort)),
			),
		),

		// Service
		fx.Provide(invitation.NewService),
	)
}
