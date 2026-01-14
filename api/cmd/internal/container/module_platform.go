package container

import (
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/platform/outbound/email"
	"github.com/team-attention/cops/api/internal/platform/outbound/email/resend"
	"github.com/team-attention/cops/api/internal/platform/outbound/email/smtp"
	"github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
	mongotx "github.com/team-attention/cops/api/internal/platform/outbound/txmanager/mongodb"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/setup/logger"
	"github.com/team-attention/cops/api/internal/platform/setup/mongodb"
	"github.com/team-attention/cops/api/internal/platform/setup/server"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Configuration (root - no dependencies)
		fx.Provide(config.InitConfig),

		// Logger (depends on config)
		fx.Provide(logger.InitLogger),

		// MongoDB (depends on config, logger)
		fx.Provide(mongodb.InitMongoDB),

		// Transaction Manager (depends on logger, mongodb)
		fx.Provide(
			fx.Annotate(
				func(l *slog.Logger, db *mongo.Database) *mongotx.MongoTransactionManager {
					return mongotx.NewMongoTransactionManager(l, db.Client())
				},
				fx.As(new(txmanager.TransactionManagerPort)),
			),
		),

		// Fiber app (depends on config, logger)
		fx.Provide(server.InitFiber),

		// Email services with factory pattern
		fx.Provide(
			fx.Annotate(
				smtp.NewSMTPEmailService,
				fx.ResultTags(`name:"smtp_email"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				resend.NewResendEmailService,
				fx.ResultTags(`name:"resend_email"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				email.NewEmailService,
				fx.ParamTags(``, ``, `name:"resend_email"`, `name:"smtp_email"`),
				fx.As(new(email.EmailServicePort)),
			),
		),
	)
}
