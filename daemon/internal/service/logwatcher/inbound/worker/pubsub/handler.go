package pubsub

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	pubsubpkg "github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
)

// LogPubsubHandler subscribes to watch target changes and updates the log service.
type LogPubsubHandler struct {
	logger       *slog.Logger
	svc          *logwatcher.Service
	targetReader pubsubpkg.ReaderPort[[]domain.WatchTarget]
	targetCh     <-chan []domain.WatchTarget
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewLogPubsubHandler creates a new pubsub handler for target watching.
func NewLogPubsubHandler(
	l *slog.Logger,
	svc *logwatcher.Service,
	targetReader pubsubpkg.ReaderPort[[]domain.WatchTarget],
) *LogPubsubHandler {
	return &LogPubsubHandler{
		logger:       l.With(slog.String("name", "log.worker.pubsub")),
		svc:          svc,
		targetReader: targetReader,
	}
}

// Name implements FsnotifyHandler interface.
func (h *LogPubsubHandler) Name() string {
	return "log-pubsub"
}

// Start implements FsnotifyHandler interface.
func (h *LogPubsubHandler) Start(ctx context.Context) error {
	h.ctx, h.cancel = context.WithCancel(ctx)
	h.targetCh = h.targetReader.Subscribe()

	h.logger.Info("log pubsub handler started")
	go h.loop()
	return nil
}

// Stop implements FsnotifyHandler interface.
func (h *LogPubsubHandler) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	if h.targetCh != nil {
		h.targetReader.Unsubscribe(h.targetCh)
	}

	h.logger.Info("log pubsub handler stopped")
	return nil
}

func (h *LogPubsubHandler) loop() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case targets := <-h.targetCh:
			// Service.UpdateTargets() → Outbound FileWatchPort.Add()/Remove()
			if err := h.svc.UpdateTargets(targets); err != nil {
				h.logger.Error("failed to update targets", slog.Any("error", err))
			}
		}
	}
}
