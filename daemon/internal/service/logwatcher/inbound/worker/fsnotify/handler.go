package fsnotify

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/repository"
)

// LogFsnotifyHandler owns the fsnotify event loop for log file watching.
type LogFsnotifyHandler struct {
	logger        *slog.Logger
	svc           *logwatcher.Service
	stateRepo     repository.StateRepositoryPort
	watcher       *fsnotify.Watcher // shared with Outbound FileWatchPort
	filePositions map[string]int64
	flushTicker   *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewLogFsnotifyHandler creates a new fsnotify handler for log watching.
func NewLogFsnotifyHandler(
	l *slog.Logger,
	svc *logwatcher.Service,
	stateRepo repository.StateRepositoryPort,
	watcher *fsnotify.Watcher,
	cfg *setup.Config,
) *LogFsnotifyHandler {
	return &LogFsnotifyHandler{
		logger:        l.With(slog.String("name", "log.worker.fsnotify")),
		svc:           svc,
		stateRepo:     stateRepo,
		watcher:       watcher,
		filePositions: make(map[string]int64),
	}
}

// Start implements FsnotifyHandler interface.
func (h *LogFsnotifyHandler) Start(ctx context.Context) error {
	h.ctx, h.cancel = context.WithCancel(context.Background())

	// Load saved file positions from SQLite
	positions, err := h.stateRepo.LoadFilePositions(ctx)
	if err != nil {
		h.logger.Warn("failed to load file positions",
			slog.Any("error", err),
		)
	} else {
		for path, pos := range positions {
			h.filePositions[path] = pos.Offset
		}
		h.logger.Info("loaded file positions",
			slog.Int("count", len(positions)),
		)
	}

	h.flushTicker = time.NewTicker(5 * time.Second)

	h.logger.Info("log fsnotify handler started")
	go h.loop()
	return nil
}

// Stop implements FsnotifyHandler interface.
func (h *LogFsnotifyHandler) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	if h.flushTicker != nil {
		h.flushTicker.Stop()
	}

	// Final flush
	if err := h.svc.Flush(ctx); err != nil {
		h.logger.Error("failed to flush on stop", slog.Any("error", err))
	}

	if err := h.stateRepo.Close(); err != nil {
		h.logger.Warn("failed to close state repository",
			slog.Any("error", err),
		)
	}

	h.logger.Info("log fsnotify handler stopped")
	return nil
}

func (h *LogFsnotifyHandler) loop() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case event := <-h.watcher.Events:
			h.handleFileEvent(event)
		case <-h.flushTicker.C:
			if err := h.svc.Flush(h.ctx); err != nil {
				h.logger.Error("failed to flush", slog.Any("error", err))
			}
		case err := <-h.watcher.Errors:
			h.logger.Error("watcher error", slog.Any("error", err))
		}
	}
}

// isValidWatchTarget checks if the directory path belongs to a registered project.
// Returns the parent project path if valid, empty string otherwise.
func (h *LogFsnotifyHandler) isValidWatchTarget(path string) string {
	// Check if path is a hidden directory (starts with '.')
	base := filepath.Base(path)
	if len(base) > 0 && base[0] == '.' {
		return ""
	}

	// Check if directory belongs to a registered project
	return h.svc.FindParentProjectPath(path)
}

// handleDirectoryCreate processes a new directory creation event.
// Handles two cases:
// 1. New directories under ~/.claude/projects/ - always watch for log files
// 2. New directories under registered project paths - add as subdirectory watch
func (h *LogFsnotifyHandler) handleDirectoryCreate(path string) {
	// Case 1: Check if this is a new directory under ~/.claude/projects/
	if h.svc.IsClaudeProjectsDir(path) {
		if err := h.svc.AddWatchForClaudeSubdir(path); err != nil {
			h.logger.Debug("failed to add watch for claude projects subdirectory",
				slog.String("path", path),
				slog.Any("error", err),
			)
		}
		return
	}

	// Case 2: Validate if directory is a valid watch target (under registered project)
	parentPath := h.isValidWatchTarget(path)
	if parentPath == "" {
		// Not registered - silently ignore
		return
	}

	// Add watch through service layer
	if err := h.svc.AddWatchForSubdirectory(path, parentPath); err != nil {
		h.logger.Debug("failed to add watch for new directory",
			slog.String("path", path),
			slog.Any("error", err),
		)
		return
	}

	h.logger.Debug("added watch for new directory",
		slog.String("path", path),
	)
	// Also scan and add any nested directories through service
	h.svc.AddNestedDirectoryWatches(path, parentPath)
}

func (h *LogFsnotifyHandler) handleFileEvent(event fsnotify.Event) {
	// Handle new directory creation
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			h.handleDirectoryCreate(event.Name)
			return
		}
	}

	// Only process JSONL files
	if !strings.HasSuffix(event.Name, ".jsonl") {
		return
	}

	// Only process write and create events
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
		return
	}

	offset := h.filePositions[event.Name]
	lines, newOffset, err := h.svc.HandleFileChange(event.Name, offset)
	if err != nil {
		h.logger.Debug("failed to handle file change",
			slog.String("path", event.Name),
			slog.Any("error", err),
		)
		return
	}

	if len(lines) > 0 {
		// Extract ClaudeDir from file path (parent directory)
		claudeDir := filepath.Dir(event.Name)
		h.svc.AddLinesForClaudeDir(claudeDir, lines)
		h.filePositions[event.Name] = newOffset

		// Save position to SQLite
		if err := h.stateRepo.SaveFilePosition(h.ctx, &domain.FilePosition{
			Path:   event.Name,
			Offset: newOffset,
		}); err != nil {
			h.logger.Warn("failed to save file position",
				slog.String("path", event.Name),
				slog.Any("error", err),
			)
		}
	}
}
