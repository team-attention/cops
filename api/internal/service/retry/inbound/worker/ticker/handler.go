package ticker

import (
	"context"
	"log/slog"
	"time"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/retry"
)

// RetryWorkerHandler handles periodic retry processing.
type RetryWorkerHandler struct {
	logger   *slog.Logger
	svc      *retry.Service
	interval time.Duration
	enabled  bool
	cancel   context.CancelFunc
}

// NewRetryWorkerHandler creates a new retry worker handler.
func NewRetryWorkerHandler(l *slog.Logger, svc *retry.Service, cfg *config.Config) *RetryWorkerHandler {
	return &RetryWorkerHandler{
		logger:   l.With(slog.String("name", "retry.worker.ticker")),
		svc:      svc,
		interval: cfg.Retry.Interval,
		enabled:  cfg.Retry.Enabled,
	}
}

// IsEnabled returns whether the retry worker is enabled.
func (h *RetryWorkerHandler) IsEnabled() bool {
	return h.enabled
}

// Start begins the retry worker. Called from fx lifecycle OnStart.
func (h *RetryWorkerHandler) Start() {
	h.logger.Info("starting retry worker",
		slog.Duration("interval", h.interval),
	)

	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel

	go h.run(ctx)
}

// Stop stops the retry worker. Called from fx lifecycle OnStop.
func (h *RetryWorkerHandler) Stop() {
	h.logger.Info("stopping retry worker")

	if h.cancel != nil {
		h.cancel()
	}
}

// run is the main worker loop.
func (h *RetryWorkerHandler) run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	h.processWithRecover(ctx)

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("retry worker stopped")
			return
		case <-ticker.C:
			h.processWithRecover(ctx)
		}
	}
}

// processWithRecover wraps batch processing with panic recovery.
func (h *RetryWorkerHandler) processWithRecover(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("panic in retry worker",
				slog.Any("panic", r),
			)
		}
	}()

	count, err := h.svc.ProcessRetryBatch(ctx)
	if err != nil {
		h.logger.Error("failed to process retry batch",
			slog.Any("error", err),
		)
		return
	}

	if count > 0 {
		h.logger.Info("batch processed",
			slog.Int("successCount", count),
		)
	}
}
