package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb"
)

func newRBACModule() fx.Option {
	return fx.Module("rbac",
		// Organization member repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationMemberRepository,
				fx.As(new(repository.OrganizationMemberRepositoryPort)),
			),
		),

		// RBAC Service (now only requires memberRepo)
		fx.Provide(rbac.NewService),
	)
}
