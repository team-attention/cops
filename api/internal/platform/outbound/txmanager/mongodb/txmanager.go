package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
)

// MongoTransactionManager implements TransactionManagerPort using MongoDB.
type MongoTransactionManager struct {
	client *mongo.Client
	logger *slog.Logger
}

// NewMongoTransactionManager creates a new MongoDB transaction manager.
func NewMongoTransactionManager(l *slog.Logger, client *mongo.Client) *MongoTransactionManager {
	return &MongoTransactionManager{
		client: client,
		logger: l.With(slog.String("name", "platform.txmanager.mongodb")),
	}
}

// WithTransaction executes a function within a MongoDB transaction.
func (m *MongoTransactionManager) WithTransaction(ctx context.Context, fn txmanager.TransactionFunc) (interface{}, error) {
	txnOpts := options.Transaction().SetWriteConcern(writeconcern.Majority())
	sessOpts := options.Session().SetDefaultTransactionOptions(txnOpts)
	session, err := m.client.StartSession(sessOpts)
	if err != nil {
		m.logger.Error("failed to start MongoDB session",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to start transaction session: %w", err)
	}
	defer session.EndSession(context.Background())

	result, err := session.WithTransaction(ctx, func(sessCtx context.Context) (interface{}, error) {
		return fn(sessCtx)
	})
	if err != nil {
		m.logger.Error("transaction failed",
			slog.Any("error", err),
		)
		return nil, err
	}
	return result, nil
}

// Interface verification
var _ txmanager.TransactionManagerPort = (*MongoTransactionManager)(nil)
