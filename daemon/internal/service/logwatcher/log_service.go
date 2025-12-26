package logwatcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/bytedance/sonic"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service contains pure business logic for log file watching and processing.
// No goroutines, no event loops - just business logic.
type Service struct {
	logger             *slog.Logger
	fileWatcher        filesystem.FileWatchPort // Outbound: fsnotify Add/Remove
	apiClient          api.APIClientPort        // Outbound: API transmission
	watchedDirs        map[string]bool
	claudeDirToProject map[string]shareddomain.ID
	bufferByProject    map[shareddomain.ID][]shareddomain.SessionRecord
	mu                 sync.Mutex
}

// NewService creates a new Log service.
func NewService(
	l *slog.Logger,
	fileWatcher filesystem.FileWatchPort,
	apiClient api.APIClientPort,
	cfg *setup.Config,
) *Service {
	return &Service{
		logger:             l.With(slog.String("name", "log.service")),
		fileWatcher:        fileWatcher,
		apiClient:          apiClient,
		watchedDirs:        make(map[string]bool),
		claudeDirToProject: make(map[string]shareddomain.ID),
		bufferByProject:    make(map[shareddomain.ID][]shareddomain.SessionRecord),
	}
}

// UpdateTargets updates the directories to watch.
// Called by Inbound (pubsub) when target configuration changes.
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build set of new directories and ProjectID mapping
	newDirs := make(map[string]bool)
	newMapping := make(map[string]shareddomain.ID)
	for _, t := range targets {
		newDirs[t.ClaudeDir] = true
		newMapping[t.ClaudeDir] = t.ProjectID
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

	// Update ClaudeDir to ProjectID mapping
	s.claudeDirToProject = newMapping

	s.logger.Info("updated watch targets",
		slog.Int("watching", len(s.watchedDirs)),
		slog.Int("mappings", len(s.claudeDirToProject)),
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

// AddRecordsForClaudeDir adds records to the buffer, associating them with the given ClaudeDir.
func (s *Service) AddRecordsForClaudeDir(claudeDir string, records []shareddomain.SessionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectID := s.claudeDirToProject[claudeDir]
	s.bufferByProject[projectID] = append(s.bufferByProject[projectID], records...)
}

// Flush sends buffered records to the API server.
// Sends separate batches for each project.
func (s *Service) Flush(ctx context.Context) error {
	s.mu.Lock()
	if len(s.bufferByProject) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Take ownership of buffer
	bufferedRecords := s.bufferByProject
	s.bufferByProject = make(map[shareddomain.ID][]shareddomain.SessionRecord)
	s.mu.Unlock()

	var totalCount int
	var lastErr error

	for projectID, records := range bufferedRecords {
		if len(records) == 0 {
			continue
		}

		batch := domain.LogBatch{
			Records:   records,
			ProjectID: projectID,
		}

		s.logger.Info("flushing log batch",
			slog.Int("count", len(records)),
			slog.String("projectID", projectID.String()),
		)

		if err := s.apiClient.SendLogs(ctx, batch); err != nil {
			// Put records back in buffer on failure
			s.mu.Lock()
			s.bufferByProject[projectID] = append(records, s.bufferByProject[projectID]...)
			s.mu.Unlock()

			lastErr = fmt.Errorf("failed to send logs for project %s: %w", projectID.String(), err)
			s.logger.Error("failed to send logs",
				slog.String("projectID", projectID.String()),
				slog.Any("error", err),
			)
			continue
		}

		totalCount += len(records)
	}

	if lastErr != nil {
		return lastErr
	}

	return nil
}

// GetProjectIDForClaudeDir returns the ProjectID for a given ClaudeDir.
// Returns empty ID if not found.
func (s *Service) GetProjectIDForClaudeDir(claudeDir string) shareddomain.ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.claudeDirToProject[claudeDir]
}
