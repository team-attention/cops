package logwatcher

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service watches Claude Code log directories for new log entries.
type Service struct {
	logger        *slog.Logger
	port          FileWatchPort
	targets       []domain.WatchTarget
	filePositions map[string]*domain.FilePosition
	onLogEntry    func(shareddomain.SessionRecord)
	watchedDirs   map[string]bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewService creates a new LogWatcher service.
func NewService(l *slog.Logger, port FileWatchPort) *Service {
	return &Service{
		logger:        l.With(slog.String("name", "logwatcher.service")),
		port:          port,
		targets:       []domain.WatchTarget{},
		filePositions: make(map[string]*domain.FilePosition),
		watchedDirs:   make(map[string]bool),
	}
}

// OnLogEntry registers a callback for new log entries.
func (s *Service) OnLogEntry(fn func(shareddomain.SessionRecord)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onLogEntry = fn
}

// UpdateTargets updates the directories to watch.
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
			if err := s.port.Remove(dir); err != nil {
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
			if err := s.port.Add(dir); err != nil {
				// Directory may not exist yet, that's OK
				s.logger.Debug("failed to add watch (directory may not exist)",
					slog.String("dir", dir),
					slog.Any("error", err),
				)
				continue
			}
			s.watchedDirs[dir] = true

			// Scan existing files in the directory
			s.scanExistingFiles(dir)
		}
	}

	s.targets = targets
	s.logger.Info("updated watch targets",
		slog.Int("watching", len(s.watchedDirs)),
	)

	return nil
}

// Start begins watching log directories.
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	go s.loop()
	s.logger.Info("log watcher started")
	return nil
}

// Stop stops watching log directories.
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
			s.handleFileEvent(event)
		case err, ok := <-s.port.Errors():
			if !ok {
				return
			}
			s.logger.Error("log watcher error", slog.Any("error", err))
		}
	}
}

func (s *Service) handleFileEvent(event FileEvent) {
	// Only process JSONL files
	if !strings.HasSuffix(event.Path, ".jsonl") {
		return
	}

	// Only process write and create events
	if !event.Has(OpWrite) && !event.Has(OpCreate) {
		return
	}

	s.readNewLines(event.Path)
}

func (s *Service) scanExistingFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		s.logger.Debug("failed to read directory",
			slog.String("dir", dir),
			slog.Any("error", err),
		)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		// Get current file size and set position to end (don't read existing content)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		s.mu.Lock()
		s.filePositions[path] = &domain.FilePosition{
			Path:   path,
			Offset: info.Size(),
		}
		s.mu.Unlock()
	}
}

func (s *Service) readNewLines(path string) {
	s.mu.Lock()
	pos, exists := s.filePositions[path]
	if !exists {
		pos = &domain.FilePosition{Path: path, Offset: 0}
		s.filePositions[path] = pos
	}
	currentOffset := pos.Offset
	s.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		s.logger.Debug("failed to open file",
			slog.String("path", path),
			slog.Any("error", err),
		)
		return
	}
	defer file.Close()

	// Seek to last known position
	if _, err := file.Seek(currentOffset, io.SeekStart); err != nil {
		s.logger.Debug("failed to seek file",
			slog.String("path", path),
			slog.Any("error", err),
		)
		return
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var newOffset int64 = currentOffset

	for scanner.Scan() {
		line := scanner.Text()
		newOffset += int64(len(line)) + 1 // +1 for newline

		if line == "" {
			continue
		}

		record, err := s.parseJSONLLine(line)
		if err != nil {
			s.logger.Debug("failed to parse JSONL line",
				slog.String("path", path),
				slog.Any("error", err),
			)
			continue
		}

		s.mu.RLock()
		onLogEntry := s.onLogEntry
		s.mu.RUnlock()

		if onLogEntry != nil {
			onLogEntry(*record)
		}
	}

	if err := scanner.Err(); err != nil {
		s.logger.Debug("error reading file",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}

	// Update file position
	s.mu.Lock()
	s.filePositions[path].Offset = newOffset
	s.mu.Unlock()
}

func (s *Service) parseJSONLLine(line string) (*shareddomain.SessionRecord, error) {
	var record shareddomain.SessionRecord
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		return nil, err
	}
	return &record, nil
}
