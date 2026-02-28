package polling

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
)

// LogPollingHandler polls the OpenCode SQLite database for new messages.
type LogPollingHandler struct {
	logger        *slog.Logger
	svc           *logwatcher.Service
	pollInterval  time.Duration
	dataDir       string
	db            *sql.DB
	lastUpdatedAt int64
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewLogPollingHandler creates a new polling handler for OpenCode.
// Receives *sql.DB from setup.InitOpenCodeDB (nil if OpenCode not installed)
// and dataDir string from setup for the OpenCode data directory.
func NewLogPollingHandler(l *slog.Logger, svc *logwatcher.Service, db *sql.DB, dataDir string) *LogPollingHandler {
	return &LogPollingHandler{
		logger:       l.With(slog.String("name", "log.worker.polling")),
		svc:          svc,
		db:           db,
		dataDir:      dataDir,
		pollInterval: 10 * time.Second,
	}
}

// Start begins the polling loop.
func (h *LogPollingHandler) Start(ctx context.Context) error {
	if h.db == nil {
		h.logger.Info("OpenCode DB not available, polling disabled")
		return nil
	}

	h.ctx, h.cancel = context.WithCancel(context.Background())
	go h.loop()
	h.logger.Info("log polling handler started")
	return nil
}

// Stop stops the polling loop.
func (h *LogPollingHandler) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	h.logger.Info("log polling handler stopped")
	return nil
}

// loop runs the periodic polling.
func (h *LogPollingHandler) loop() {
	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	// Perform initial poll immediately
	h.pollNewMessages()

	for {
		select {
		case <-ticker.C:
			h.pollNewMessages()
		case <-h.ctx.Done():
			return
		}
	}
}

// openCodeMessage represents a row from the OpenCode message table
// joined with the session table for project directory.
// JSON tags use camelCase per project convention (go-platform-domain.md).
type openCodeMessage struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionId"`
	TimeCreated int64  `json:"timeCreated"`
	TimeUpdated int64  `json:"timeUpdated"`
	Data        string `json:"data"`
	ProjectPath string `json:"projectPath,omitempty"`
}

// pollNewMessages queries the OpenCode SQLite DB for messages newer than lastUpdatedAt.
// Groups messages by project path for per-project resolution.
func (h *LogPollingHandler) pollNewMessages() {
	rows, err := h.db.QueryContext(h.ctx,
		`SELECT m.id, m.session_id, m.time_created, m.time_updated, m.data,
		        s.directory as project_path
		FROM message m
		LEFT JOIN session s ON m.session_id = s.id
		WHERE m.time_updated > ? ORDER BY m.time_updated ASC`,
		h.lastUpdatedAt,
	)
	if err != nil {
		h.logger.Warn("failed to query OpenCode messages",
			slog.Any("error", err),
		)
		return
	}
	defer rows.Close()

	linesByProject := make(map[string][]string)
	var maxUpdatedAt int64

	for rows.Next() {
		var msg openCodeMessage
		var projectPath sql.NullString

		if err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.TimeCreated,
			&msg.TimeUpdated,
			&msg.Data,
			&projectPath,
		); err != nil {
			h.logger.Warn("failed to scan OpenCode message row",
				slog.Any("error", err),
			)
			continue
		}

		if projectPath.Valid {
			msg.ProjectPath = projectPath.String
		}

		data, err := json.Marshal(msg)
		if err != nil {
			h.logger.Warn("failed to marshal OpenCode message",
				slog.Any("error", err),
			)
			continue
		}

		linesByProject[msg.ProjectPath] = append(linesByProject[msg.ProjectPath], string(data))

		if msg.TimeUpdated > maxUpdatedAt {
			maxUpdatedAt = msg.TimeUpdated
		}
	}

	if err := rows.Err(); err != nil {
		h.logger.Warn("error iterating OpenCode message rows",
			slog.Any("error", err),
		)
	}

	for projectPath, lines := range linesByProject {
		if projectPath == "" {
			h.logger.Debug("OpenCode message without project path, skipping",
				slog.Int("lineCount", len(lines)),
			)
			continue
		}

		compositeKey := h.svc.ResolveAndCacheOpenCodeProject(h.dataDir, projectPath)
		h.svc.AddLinesForDir(compositeKey, lines, domain.ProviderOpenCode)
	}

	if maxUpdatedAt > 0 {
		h.lastUpdatedAt = maxUpdatedAt
	}
}
