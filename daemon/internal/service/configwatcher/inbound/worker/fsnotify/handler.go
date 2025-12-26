package fsnotify

import (
	"context"
	"log/slog"

	"github.com/fsnotify/fsnotify"

	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher"
)

// ConfigWatcherFsnotifyHandler owns the fsnotify event loop for config watching.
type ConfigWatcherFsnotifyHandler struct {
	logger     *slog.Logger
	svc        *configwatcher.Service
	watcher    *fsnotify.Watcher
	configPath string
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewConfigWatcherFsnotifyHandler creates a new fsnotify handler for config watching.
func NewConfigWatcherFsnotifyHandler(
	l *slog.Logger,
	svc *configwatcher.Service,
	cfg *setup.Config,
) (*ConfigWatcherFsnotifyHandler, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &ConfigWatcherFsnotifyHandler{
		logger:     l.With(slog.String("name", "configwatcher.worker.fsnotify")),
		svc:        svc,
		watcher:    watcher,
		configPath: cfg.Cops.GlobalConfigPath,
	}, nil
}

// Start implements FsnotifyHandler interface.
func (h *ConfigWatcherFsnotifyHandler) Start(ctx context.Context) error {
	h.ctx, h.cancel = context.WithCancel(context.Background())

	// Initial config load - restore watches from saved config
	h.logger.Info("loading initial config for watch restoration", slog.String("path", h.configPath))
	if err := h.svc.HandleConfigChange(h.configPath); err != nil {
		h.logger.Warn("initial config load failed", slog.Any("error", err))
	} else {
		h.logger.Info("initial config loaded, watch targets published")
	}

	// Watch config file
	if err := h.watcher.Add(h.configPath); err != nil {
		return err
	}

	h.logger.Info("config watcher started", slog.String("path", h.configPath))

	go h.loop()
	return nil
}

// Stop implements FsnotifyHandler interface.
func (h *ConfigWatcherFsnotifyHandler) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	return h.watcher.Close()
}

func (h *ConfigWatcherFsnotifyHandler) loop() {
	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("config watcher stopping")
			return
		case event := <-h.watcher.Events:
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if err := h.svc.HandleConfigChange(h.configPath); err != nil {
					h.logger.Error("config change handling failed", slog.Any("error", err))
				}
			}
		case err := <-h.watcher.Errors:
			h.logger.Error("watcher error", slog.Any("error", err))
		}
	}
}
