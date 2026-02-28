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

// openCodeMessage represents a row from the OpenCode messages table.
type openCodeMessage struct {
	ID         string  `json:"id"`
	SessionID  string  `json:"session_id"`
	Role       string  `json:"role"`
	Parts      string  `json:"parts"`
	Model      string  `json:"model"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
	FinishedAt *int64  `json:"finished_at,omitempty"`
}

// pollNewMessages queries the OpenCode SQLite DB for messages newer than lastUpdatedAt.
func (h *LogPollingHandler) pollNewMessages() {
	rows, err := h.db.QueryContext(h.ctx,
		`SELECT id, session_id, role, parts, model, created_at, updated_at, finished_at
		FROM messages WHERE updated_at > ? ORDER BY updated_at ASC`,
		h.lastUpdatedAt,
	)
	if err != nil {
		h.logger.Warn("failed to query OpenCode messages",
			slog.Any("error", err),
		)
		return
	}
	defer rows.Close()

	var lines []string
	var maxUpdatedAt int64

	for rows.Next() {
		var msg openCodeMessage
		var finishedAt sql.NullInt64

		if err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Parts,
			&msg.Model,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&finishedAt,
		); err != nil {
			h.logger.Warn("failed to scan OpenCode message row",
				slog.Any("error", err),
			)
			continue
		}

		if finishedAt.Valid {
			msg.FinishedAt = &finishedAt.Int64
		}

		data, err := json.Marshal(msg)
		if err != nil {
			h.logger.Warn("failed to marshal OpenCode message",
				slog.Any("error", err),
			)
			continue
		}

		lines = append(lines, string(data))

		if msg.UpdatedAt > maxUpdatedAt {
			maxUpdatedAt = msg.UpdatedAt
		}
	}

	if err := rows.Err(); err != nil {
		h.logger.Warn("error iterating OpenCode message rows",
			slog.Any("error", err),
		)
	}

	if len(lines) > 0 {
		h.svc.AddLinesForDir(h.dataDir, lines, domain.ProviderOpenCode)
		h.lastUpdatedAt = maxUpdatedAt
	}
}
