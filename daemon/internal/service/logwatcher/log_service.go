package logwatcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

const (
	// minBatchSize is the minimum number of lines to attempt sending.
	minBatchSize = 1
)

// Service contains pure business logic for log file watching and processing.
// No goroutines, no event loops - just business logic.
type Service struct {
	logger             *slog.Logger
	fileWatcher        filesystem.FileWatchPort // Outbound: fsnotify Add/Remove
	apiClient          api.APIClientPort        // Outbound: API transmission
	maxBatchSize       int                      // Maximum lines per batch (from config)
	watchedDirs        map[string]bool
	claudeDirToProject map[string]shareddomain.ID
	projectPathToID    map[string]shareddomain.ID // ProjectPath -> ProjectID mapping for hierarchical matching
	bufferByProject    map[shareddomain.ID][]string
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
		maxBatchSize:       cfg.Cops.MaxBatchSize,
		watchedDirs:        make(map[string]bool),
		claudeDirToProject: make(map[string]shareddomain.ID),
		projectPathToID:    make(map[string]shareddomain.ID),
		bufferByProject:    make(map[shareddomain.ID][]string),
	}
}

// UpdateTargets updates the directories to watch.
// Called by Inbound (pubsub) when target configuration changes.
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build set of new directories and ProjectID mappings
	newDirs := make(map[string]bool)
	newClaudeDirMapping := make(map[string]shareddomain.ID)
	newProjectPathMapping := make(map[string]shareddomain.ID)

	for _, t := range targets {
		newDirs[t.ClaudeDir] = true
		newClaudeDirMapping[t.ClaudeDir] = t.ProjectID
		newProjectPathMapping[t.ProjectPath] = t.ProjectID
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

	// Update mappings
	s.claudeDirToProject = newClaudeDirMapping
	s.projectPathToID = newProjectPathMapping

	s.logger.Info("updated watch targets",
		slog.Int("watching", len(s.watchedDirs)),
		slog.Int("claudeDirMappings", len(s.claudeDirToProject)),
		slog.Int("projectPathMappings", len(s.projectPathToID)),
	)

	return nil
}

// HandleFileChange handles a file change event and returns raw JSONL lines.
// Called by Inbound (fsnotify) when a log file is modified.
func (s *Service) HandleFileChange(path string, fromOffset int64) ([]string, int64, error) {
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

	var lines []string
	newOffset := fromOffset

	for scanner.Scan() {
		line := scanner.Text()
		newOffset += int64(len(line)) + 1 // +1 for newline

		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return lines, newOffset, fmt.Errorf("error reading file: %w", err)
	}

	return lines, newOffset, nil
}

// AddLinesForClaudeDir adds raw JSONL lines to the buffer, associating them with the given ClaudeDir.
func (s *Service) AddLinesForClaudeDir(claudeDir string, lines []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectID := s.claudeDirToProject[claudeDir]
	s.bufferByProject[projectID] = append(s.bufferByProject[projectID], lines...)
}

// Flush sends buffered lines to the API server using adaptive batching.
// Sends separate batches for each project with dynamic size adjustment.
func (s *Service) Flush(ctx context.Context) error {
	// 1. Lock mutex and check if buffer is empty
	s.mu.Lock()
	if len(s.bufferByProject) == 0 {
		s.mu.Unlock()
		return nil
	}

	// 2. Take ownership of buffer (swap with new empty map) and unlock
	bufferedLines := s.bufferByProject
	s.bufferByProject = make(map[shareddomain.ID][]string)
	s.mu.Unlock()

	// 3. Initialize lastErr variable for tracking errors
	var lastErr error

	// 4. For each projectID and lines in bufferedLines
	for projectID, lines := range bufferedLines {
		// a. Skip if lines is empty
		if len(lines) == 0 {
			continue
		}

		// b. Log start of flush
		s.logger.Info("flushing log batch",
			slog.String("projectID", projectID.String()),
			slog.Int("totalLines", len(lines)),
			slog.Int("maxBatchSize", s.maxBatchSize),
		)

		// c. Call s.flushProjectLines
		if err := s.flushProjectLines(ctx, projectID, lines); err != nil {
			// d. If error returned, set lastErr and continue to next project
			lastErr = err
			continue
		}
	}

	// 5. Return lastErr (nil if all projects succeeded)
	return lastErr
}

// flushProjectLines sends all lines for a single project using adaptive batching.
// Returns any unsent lines to the buffer on failure.
func (s *Service) flushProjectLines(ctx context.Context, projectID shareddomain.ID, lines []string) error {
	// 1. Initialize state
	currentBatchSize := s.maxBatchSize
	remainingLines := lines
	batchNum := 0
	totalSent := 0

	// 2. While len(remainingLines) > 0
	for len(remainingLines) > 0 {
		// a. Increment batchNum
		batchNum++

		// b. Calculate batchSize = min(currentBatchSize, len(remainingLines))
		batchSize := currentBatchSize
		if batchSize > len(remainingLines) {
			batchSize = len(remainingLines)
		}

		// c. Extract currentBatch (make a copy to avoid aliasing issues)
		currentBatch := make([]string, batchSize)
		copy(currentBatch, remainingLines[:batchSize])

		// d. Create domain.LogBatch
		batch := domain.LogBatch{
			Lines:     currentBatch,
			ProjectID: projectID,
		}

		s.logger.Debug("attempting batch",
			slog.String("projectID", projectID.String()),
			slog.Int("batchNum", batchNum),
			slog.Int("batchSize", batchSize),
			slog.Int("remaining", len(remainingLines)),
			slog.Int("currentBatchSize", currentBatchSize),
		)

		// e. Call s.sendBatchWithRetry
		if err := s.sendBatchWithRetry(ctx, batch, projectID, batchNum, &currentBatchSize); err != nil {
			// f. If error
			if errutil.IsPayloadTooLarge(err) && currentBatchSize == minBatchSize {
				// Single line too large, skip it
				s.logger.Error("single line too large, skipping",
					slog.String("projectID", projectID.String()),
					slog.Int("batchNum", batchNum),
				)
				// Remove the first line from remainingLines
				remainingLines = remainingLines[1:]
				// Reset currentBatchSize to s.maxBatchSize
				currentBatchSize = s.maxBatchSize
				continue
			}
			// Other errors - return lines to buffer and return error
			s.returnLinesToBuffer(projectID, remainingLines)
			return err
		}

		// After sendBatchWithRetry, batch.Lines may have been reduced
		// Use the actual length that was sent
		actualSentSize := len(batch.Lines)

		s.logger.Debug("batch sent successfully",
			slog.String("projectID", projectID.String()),
			slog.Int("batchNum", batchNum),
			slog.Int("actualSentSize", actualSentSize),
			slog.Int("remainingBefore", len(remainingLines)),
		)

		// g. On success
		// Remove sent lines from remainingLines
		remainingLines = remainingLines[actualSentSize:]
		// totalSent += actualSentSize
		totalSent += actualSentSize

		s.logger.Debug("after removing sent lines",
			slog.String("projectID", projectID.String()),
			slog.Int("batchNum", batchNum),
			slog.Int("remainingAfter", len(remainingLines)),
			slog.Int("totalSent", totalSent),
		)

		// Double currentBatchSize (capped at s.maxBatchSize)
		currentBatchSize = currentBatchSize * 2
		if currentBatchSize > s.maxBatchSize {
			currentBatchSize = s.maxBatchSize
		}
	}

	// 3. Log completion
	s.logger.Info("flush completed",
		slog.String("projectID", projectID.String()),
		slog.Int("totalBatches", batchNum),
		slog.Int("totalLinesSent", totalSent),
	)

	// 4. Return nil
	return nil
}

// sendBatchWithRetry attempts to send a batch, retrying with reduced size on 413 errors.
// Updates currentBatchSize pointer on 413 errors.
// Returns nil on success, error on failure.
func (s *Service) sendBatchWithRetry(
	ctx context.Context,
	batch domain.LogBatch,
	projectID shareddomain.ID,
	batchNum int,
	currentBatchSize *int,
) error {
	// 1. Loop (infinite loop, will break on success or non-413 error)
	for {
		// a. Call s.apiClient.SendLogs
		err := s.apiClient.SendLogs(ctx, batch)

		// b. If no error
		if err == nil {
			// Log success
			s.logger.Debug("batch sent successfully",
				slog.String("projectID", projectID.String()),
				slog.Int("batchNum", batchNum),
				slog.Int("linesInBatch", len(batch.Lines)),
				slog.Int("currentBatchSize", *currentBatchSize),
			)
			return nil
		}

		// c. If errutil.IsPayloadTooLarge(err)
		if errutil.IsPayloadTooLarge(err) {
			// Calculate newSize
			newSize := *currentBatchSize / 2
			if newSize < minBatchSize {
				newSize = minBatchSize
			}

			// Log warning
			s.logger.Warn("batch too large, reducing size",
				slog.String("projectID", projectID.String()),
				slog.Int("batchNum", batchNum),
				slog.Int("previousSize", *currentBatchSize),
				slog.Int("newSize", newSize),
			)

			// If newSize == *currentBatchSize (already at minimum)
			if newSize == *currentBatchSize {
				// We're already at minimum size (1 line), return error
				return err
			}

			// Update *currentBatchSize
			*currentBatchSize = newSize

			// Resize batch.Lines to newSize (but not larger than current batch)
			if newSize > len(batch.Lines) {
				newSize = len(batch.Lines)
			}
			batch.Lines = batch.Lines[:newSize]

			// Continue loop (retry with smaller batch)
			continue
		}

		// d. For other errors
		s.logger.Error("failed to send batch",
			slog.String("projectID", projectID.String()),
			slog.Int("batchNum", batchNum),
			slog.Any("error", err),
		)
		return err
	}
}

// returnLinesToBuffer puts unsent lines back into the buffer for retry in next flush.
func (s *Service) returnLinesToBuffer(projectID shareddomain.ID, lines []string) {
	// 1. Lock mutex
	s.mu.Lock()
	defer s.mu.Unlock()

	// 2. Prepend lines to s.bufferByProject[projectID]
	s.bufferByProject[projectID] = append(lines, s.bufferByProject[projectID]...)

	// 3. Log info
	s.logger.Info("returned lines to buffer",
		slog.String("projectID", projectID.String()),
		slog.Int("lineCount", len(lines)),
	)
}

// GetProjectIDForClaudeDir returns the ProjectID for a given ClaudeDir.
// Uses hierarchical matching: first tries exact match, then finds longest path prefix.
// Returns empty ID if not found.
func (s *Service) GetProjectIDForClaudeDir(claudeDir string) shareddomain.ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try exact match first
	if projectID, ok := s.claudeDirToProject[claudeDir]; ok {
		return projectID
	}

	// No exact match - try hierarchical matching
	// Decode claudeDir to original path
	decodedPath := pathutil.DecodeClaudeProjectDir(claudeDir)
	if decodedPath == "" {
		// Invalid Claude directory format
		return ""
	}

	// Find all matching parent paths
	var longestMatch string
	var matchedID shareddomain.ID

	for projectPath, projectID := range s.projectPathToID {
		// Check if projectPath is a prefix of decodedPath
		// Must be exact match OR decodedPath must start with projectPath followed by separator
		if projectPath == decodedPath {
			// Exact match (already checked above, but keep for completeness)
			return projectID
		}

		// Check if it's a proper prefix (with separator)
		if strings.HasPrefix(decodedPath, projectPath+"/") {
			// This is a parent directory match
			// Keep the longest (most specific) match
			if len(projectPath) > len(longestMatch) {
				longestMatch = projectPath
				matchedID = projectID
			}
		}
	}

	return matchedID
}
