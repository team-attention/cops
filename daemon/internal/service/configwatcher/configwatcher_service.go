package configwatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
)

// Service watches the global config file for changes.
type Service struct {
	logger   *slog.Logger
	port     FileWatchPort
	path     string
	onChange func(domain.GlobalConfig)
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewService creates a new ConfigWatcher service.
func NewService(l *slog.Logger, port FileWatchPort, cfg *config.Config) *Service {
	return &Service{
		logger: l.With(slog.String("name", "configwatcher.service")),
		port:   port,
		path:   cfg.Cops.GlobalConfigPath,
	}
}

// OnConfigChange registers a callback for config changes.
func (s *Service) OnConfigChange(fn func(domain.GlobalConfig)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = fn
}

// LoadConfig loads and parses the global config file.
func (s *Service) LoadConfig() (*domain.GlobalConfig, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &domain.GlobalConfig{Projects: []domain.ProjectConfig{}}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg domain.GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Start begins watching the config file.
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if err := s.port.Watch(s.path); err != nil {
		s.logger.Warn("failed to watch config file, will retry on next change",
			slog.String("path", s.path),
			slog.Any("error", err),
		)
	}

	go s.loop()
	s.logger.Info("config watcher started", slog.String("path", s.path))
	return nil
}

// Stop stops watching the config file.
func (s *Service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	return s.port.Close()
}

func (s *Service) loop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event, ok := <-s.port.Events():
			if !ok {
				return
			}
			if event.Has(OpWrite) || event.Has(OpCreate) {
				s.handleConfigChange()
			}
		case err, ok := <-s.port.Errors():
			if !ok {
				return
			}
			s.logger.Error("config watcher error", slog.Any("error", err))
		}
	}
}

func (s *Service) handleConfigChange() {
	cfg, err := s.LoadConfig()
	if err != nil {
		s.logger.Error("failed to load config after change", slog.Any("error", err))
		return
	}

	s.mu.RLock()
	onChange := s.onChange
	s.mu.RUnlock()

	if onChange != nil {
		s.logger.Info("config changed, notifying listeners",
			slog.Int("projects", len(cfg.Projects)),
		)
		onChange(*cfg)
	}
}
