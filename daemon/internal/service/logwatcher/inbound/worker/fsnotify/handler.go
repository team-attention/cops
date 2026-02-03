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

	// Perform initial scan of existing files after event loop starts
	// This ensures any file changes during scan are captured by fsnotify
	h.scanExistingFiles()

	return nil
}

// scanExistingFiles scans all watched directories for existing JSONL files
// and processes any new content since the last recorded position.
// This must be called after fsnotify watches are set up to avoid missing
// events that occur during the scan.
func (h *LogFsnotifyHandler) scanExistingFiles() {
	watchList := h.watcher.WatchList()
	if len(watchList) == 0 {
		h.logger.Info("no watched directories, skipping initial scan")
		return
	}

	h.logger.Info("starting initial scan of existing files",
		slog.Int("watchedDirs", len(watchList)),
	)

	var totalFiles int

	for _, dir := range watchList {
		// Find all .jsonl files in the directory
		pattern := filepath.Join(dir, "*.jsonl")
		files, err := filepath.Glob(pattern)
		if err != nil {
			h.logger.Warn("failed to glob directory",
				slog.String("dir", dir),
				slog.Any("error", err),
			)
			continue
		}

		for _, filePath := range files {
			// Get the stored offset (0 if new file)
			offset := h.filePositions[filePath]

			// Get current file size to check if file has new content
			info, err := os.Stat(filePath)
			if err != nil {
				h.logger.Debug("failed to stat file during scan",
					slog.String("path", filePath),
					slog.Any("error", err),
				)
				continue
			}

			// Skip if file hasn't grown since last position
			if info.Size() <= offset {
				continue
			}

			// Process the file from the stored offset
			h.processFileFromOffset(filePath, offset)
			totalFiles++
		}
	}

	h.logger.Info("initial scan completed",
		slog.Int("filesProcessed", totalFiles),
	)
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
		// Recursively watch subdirectories and scan existing files
		// This handles the race condition where files are created before the watch is added
		h.scanDirectoryRecursive(path)
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

	// Get stored offset (0 if not found for new files)
	offset := h.filePositions[event.Name]
	h.processFileFromOffset(event.Name, offset)
}

// scanDirectoryRecursive adds watches for a directory and all its subdirectories,
// then scans for existing .jsonl files. This handles the race condition where
// subdirectories and files are created before the parent directory watch is established.
func (h *LogFsnotifyHandler) scanDirectoryRecursive(root string) {
	// Walk all subdirectories and add watches
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}
		if d.IsDir() {
			// Skip hidden directories
			if len(d.Name()) > 0 && d.Name()[0] == '.' {
				return filepath.SkipDir
			}
			// Add watch for this directory (skip root, already added)
			if path != root {
				if err := h.svc.AddWatchForClaudeSubdir(path); err != nil {
					h.logger.Debug("failed to add watch during recursive scan",
						slog.String("path", path),
						slog.Any("error", err),
					)
				}
			}
		}
		return nil
	})
	if err != nil {
		h.logger.Debug("failed to walk directory for watches",
			slog.String("root", root),
			slog.Any("error", err),
		)
	}

	// Scan for .jsonl files in all directories
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".jsonl") {
			offset := h.filePositions[path]
			h.processFileFromOffset(path, offset)
		}
		return nil
	})
	if err != nil {
		h.logger.Debug("failed to scan files in directory",
			slog.String("root", root),
			slog.Any("error", err),
		)
	}
}

// processFileFromOffset reads and processes a single file from the given offset.
// This is a shared helper used by both initial scan and fsnotify event handling.
func (h *LogFsnotifyHandler) processFileFromOffset(filePath string, offset int64) {
	lines, newOffset, err := h.svc.HandleFileChange(filePath, offset)
	if err != nil {
		h.logger.Debug("failed to handle file change",
			slog.String("path", filePath),
			slog.Any("error", err),
		)
		return
	}

	if len(lines) == 0 {
		return
	}

	// Determine claudeDir based on file location
	var claudeDir string
	if strings.Contains(filePath, "/subagents/") {
		// SubAgent: {claudeDir}/{sessionId}/subagents/agent-{agentId}.jsonl
		subagentsDir := filepath.Dir(filePath)
		sessionDir := filepath.Dir(subagentsDir)
		claudeDir = filepath.Dir(sessionDir)
	} else {
		claudeDir = filepath.Dir(filePath)
	}

	h.svc.AddLinesForClaudeDir(claudeDir, lines)

	h.filePositions[filePath] = newOffset

	if err := h.stateRepo.SaveFilePosition(h.ctx, &domain.FilePosition{
		Path:   filePath,
		Offset: newOffset,
	}); err != nil {
		h.logger.Warn("failed to save file position",
			slog.String("path", filePath),
			slog.Any("error", err),
		)
	}
}
