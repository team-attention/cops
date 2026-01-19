package setup

import (
	"log/slog"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub/inmemory"
)

// InitTargetPubSub creates a pub/sub instance for watch target distribution.
func InitTargetPubSub(l *slog.Logger) *inmemory.PubSub[[]domain.WatchTarget] {
	ps := inmemory.NewPubSub[[]domain.WatchTarget](l, 10)
	l.Info("target pubsub initialized")
	return ps
}
