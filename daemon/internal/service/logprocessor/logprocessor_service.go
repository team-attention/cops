package logprocessor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service buffers log entries and sends them to the API server.
type Service struct {
	logger        *slog.Logger
	apiPort       APIClientPort
	buffer        []shareddomain.SessionRecord
	flushInterval time.Duration
	debug         bool
	daemonID      string
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewService creates a new LogProcessor service.
func NewService(l *slog.Logger, port APIClientPort, cfg *config.Config) *Service {
	return &Service{
		logger:        l.With(slog.String("name", "logprocessor.service")),
		apiPort:       port,
		buffer:        []shareddomain.SessionRecord{},
		flushInterval: cfg.Cops.FlushInterval,
		debug:         cfg.App.Debug,
		daemonID:      uuid.New().String(),
	}
}

// AddEntry adds a log entry to the buffer.
func (s *Service) AddEntry(entry shareddomain.SessionRecord) {
	s.mu.Lock()
	s.buffer = append(s.buffer, entry)
	bufferSize := len(s.buffer)
	s.mu.Unlock()

	s.logger.Debug("added entry to buffer",
		slog.String("uuid", entry.UUID),
		slog.Int("bufferSize", bufferSize),
	)

	// In debug mode, flush immediately
	if s.debug {
		if err := s.Flush(s.ctx); err != nil {
			s.logger.Error("failed to flush in debug mode", slog.Any("error", err))
		}
	}
}

// Flush sends buffered entries to the API server.
func (s *Service) Flush(ctx context.Context) error {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Take ownership of buffer
	records := s.buffer
	s.buffer = []shareddomain.SessionRecord{}
	s.mu.Unlock()

	batch := domain.LogBatch{
		Records:   records,
		DaemonID:  s.daemonID,
		CreatedAt: time.Now(),
	}

	s.logger.Info("flushing log batch",
		slog.Int("count", len(records)),
	)

	if err := s.apiPort.SendLogs(ctx, batch); err != nil {
		// Put records back in buffer on failure
		s.mu.Lock()
		s.buffer = append(records, s.buffer...)
		s.mu.Unlock()

		return err
	}

	return nil
}

// Start begins the flush ticker.
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if !s.debug {
		go s.flushLoop()
	}

	s.logger.Info("log processor started",
		slog.Bool("debug", s.debug),
		slog.Duration("flushInterval", s.flushInterval),
	)
	return nil
}

// Stop flushes remaining buffer and stops the service.
func (s *Service) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	// Final flush
	if err := s.Flush(ctx); err != nil {
		s.logger.Error("failed to flush on stop", slog.Any("error", err))
	}

	s.logger.Info("log processor stopped")
	return nil
}

func (s *Service) flushLoop() {
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.Flush(s.ctx); err != nil {
				s.logger.Error("failed to flush", slog.Any("error", err))
			}
		}
	}
}
