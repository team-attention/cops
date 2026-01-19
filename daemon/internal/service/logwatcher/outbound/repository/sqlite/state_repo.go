package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/repository"
)

// SQLiteStateRepository implements StateRepositoryPort using SQLite.
type SQLiteStateRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSQLiteStateRepository creates a new SQLite state repository.
func NewSQLiteStateRepository(l *slog.Logger, db *sql.DB) (repository.StateRepositoryPort, error) {
	logger := l.With(slog.String("name", "log.repository.sqlite"))
	logger.Info("SQLite state repository initialized")

	return &SQLiteStateRepository{
		db:     db,
		logger: logger,
	}, nil
}

// LoadFilePositions loads all file positions from the database.
func (r *SQLiteStateRepository) LoadFilePositions(ctx context.Context) (map[string]*domain.FilePosition, error) {
	query := `SELECT path, offset, updated_at FROM file_positions`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query file positions: %w", err)
	}
	defer rows.Close()

	positions := make(map[string]*domain.FilePosition)
	for rows.Next() {
		var path string
		var offset int64
		var updatedAt string

		if err := rows.Scan(&path, &offset, &updatedAt); err != nil {
			r.logger.Warn("failed to scan file position",
				slog.Any("error", err),
			)
			continue
		}

		parsedTime, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			r.logger.Warn("failed to parse updated_at",
				slog.String("path", path),
				slog.Any("error", err),
			)
			parsedTime = time.Now()
		}

		positions[path] = &domain.FilePosition{
			Path:      path,
			Offset:    offset,
			UpdatedAt: parsedTime,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file positions: %w", err)
	}

	r.logger.Debug("loaded file positions",
		slog.Int("count", len(positions)),
	)

	return positions, nil
}

// SaveFilePosition saves a file position to the database.
func (r *SQLiteStateRepository) SaveFilePosition(ctx context.Context, pos *domain.FilePosition) error {
	query := `
		INSERT INTO file_positions (path, offset, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			offset = excluded.offset,
			updated_at = excluded.updated_at
	`

	updatedAt := time.Now().Format(time.RFC3339)
	if _, err := r.db.ExecContext(ctx, query, pos.Path, pos.Offset, updatedAt); err != nil {
		return fmt.Errorf("failed to save file position: %w", err)
	}

	return nil
}

// DeleteFilePosition deletes a file position from the database.
func (r *SQLiteStateRepository) DeleteFilePosition(ctx context.Context, path string) error {
	query := `DELETE FROM file_positions WHERE path = ?`
	if _, err := r.db.ExecContext(ctx, query, path); err != nil {
		return fmt.Errorf("failed to delete file position: %w", err)
	}

	r.logger.Debug("deleted file position",
		slog.String("path", path),
	)

	return nil
}

// Close closes the database connection.
func (r *SQLiteStateRepository) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}
