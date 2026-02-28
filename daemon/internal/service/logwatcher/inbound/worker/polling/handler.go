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

// openCodeMessage represents a row from the OpenCode messages table
// joined with the sessions table for project path.
// JSON tags use camelCase per project convention (go-platform-domain.md).
type openCodeMessage struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionId"`
	Role        string `json:"role"`
	Parts       string `json:"parts"`
	Model       string `json:"model"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
	FinishedAt  *int64 `json:"finishedAt,omitempty"`
	ProjectPath string `json:"projectPath,omitempty"`
}

// pollNewMessages queries the OpenCode SQLite DB for messages newer than lastUpdatedAt.
// Groups messages by project path for per-project resolution.
func (h *LogPollingHandler) pollNewMessages() {
	rows, err := h.db.QueryContext(h.ctx,
		`SELECT m.id, m.session_id, m.role, m.parts, m.model,
		        m.created_at, m.updated_at, m.finished_at,
		        s.dir as project_path
		FROM messages m
		LEFT JOIN sessions s ON m.session_id = s.id
		WHERE m.updated_at > ? ORDER BY m.updated_at ASC`,
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
		var finishedAt sql.NullInt64
		var projectPath sql.NullString

		if err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Parts,
			&msg.Model,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&finishedAt,
			&projectPath,
		); err != nil {
			h.logger.Warn("failed to scan OpenCode message row",
				slog.Any("error", err),
			)
			continue
		}

		if finishedAt.Valid {
			msg.FinishedAt = &finishedAt.Int64
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

		if msg.UpdatedAt > maxUpdatedAt {
			maxUpdatedAt = msg.UpdatedAt
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
