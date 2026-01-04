package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/organization"
	"github.com/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository/mongodb"
)

func newOrganizationModule() fx.Option {
	return fx.Module("organization",
		// Organization repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationRepository,
				fx.As(new(repository.OrganizationRepositoryPort)),
			),
		),

		// Service
		fx.Provide(organization.NewService),

		// ConnectRPC handler (private - requires auth)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewOrganizationGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
