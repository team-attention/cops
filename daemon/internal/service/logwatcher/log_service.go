package logwatcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service contains pure business logic for log file watching and processing.
// No goroutines, no event loops - just business logic.
type Service struct {
	logger      *slog.Logger
	fileWatcher filesystem.FileWatchPort // Outbound: fsnotify Add/Remove
	apiClient   api.APIClientPort        // Outbound: API transmission
	watchedDirs map[string]bool
	buffer      []shareddomain.SessionRecord
	daemonID    string
	mu          sync.Mutex
}

// NewService creates a new Log service.
func NewService(
	l *slog.Logger,
	fileWatcher filesystem.FileWatchPort,
	apiClient api.APIClientPort,
	cfg *setup.Config,
) *Service {
	return &Service{
		logger:      l.With(slog.String("name", "log.service")),
		fileWatcher: fileWatcher,
		apiClient:   apiClient,
		watchedDirs: make(map[string]bool),
		buffer:      []shareddomain.SessionRecord{},
		daemonID:    uuid.New().String(),
	}
}

// UpdateTargets updates the directories to watch.
// Called by Inbound (pubsub) when target configuration changes.
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build set of new directories
	newDirs := make(map[string]bool)
	for _, t := range targets {
		newDirs[t.ClaudeDir] = true
	}

	// Remove watches for directories no longer in targets
	for dir := range s.watchedDirs {
		if !newDirs[dir] {
			if err := s.fileWatcher.Remove(dir); err != nil {
				s.logger.Warn("failed to remove watch",
					slog.String("dir", dir),
					slog.Any("error", err),
				)
			}
			delete(s.watchedDirs, dir)
		}
	}

	// Add watches for new directories
	for dir := range newDirs {
		if !s.watchedDirs[dir] {
			if err := s.fileWatcher.Add(dir); err != nil {
				// Directory may not exist yet, that's OK
				s.logger.Debug("failed to add watch (directory may not exist)",
					slog.String("dir", dir),
					slog.Any("error", err),
				)
				continue
			}
			s.watchedDirs[dir] = true
		}
	}

	s.logger.Info("updated watch targets",
		slog.Int("watching", len(s.watchedDirs)),
	)

	return nil
}

// HandleFileChange handles a file change event and returns parsed records.
// Called by Inbound (fsnotify) when a log file is modified.
func (s *Service) HandleFileChange(path string, fromOffset int64) ([]shareddomain.SessionRecord, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fromOffset, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to last known position
	if _, err := file.Seek(fromOffset, io.SeekStart); err != nil {
		return nil, fromOffset, fmt.Errorf("failed to seek file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var records []shareddomain.SessionRecord
	newOffset := fromOffset

	for scanner.Scan() {
		line := scanner.Text()
		newOffset += int64(len(line)) + 1 // +1 for newline

		if line == "" {
			continue
		}

		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			s.logger.Debug("failed to parse JSONL line",
				slog.String("path", path),
				slog.Any("error", err),
			)
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return records, newOffset, fmt.Errorf("error reading file: %w", err)
	}

	return records, newOffset, nil
}

// AddRecords adds records to the buffer.
func (s *Service) AddRecords(records []shareddomain.SessionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer = append(s.buffer, records...)
}

// Flush sends buffered records to the API server.
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

	if err := s.apiClient.SendLogs(ctx, batch); err != nil {
		// Put records back in buffer on failure
		s.mu.Lock()
		s.buffer = append(records, s.buffer...)
		s.mu.Unlock()

		return fmt.Errorf("failed to send logs: %w", err)
	}

	return nil
}
