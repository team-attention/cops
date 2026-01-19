package setup

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitSQLite initializes SQLite database connection with WAL mode and creates tables.
func InitSQLite(paths *ExpandedPaths, logger *slog.Logger) (*sql.DB, error) {
	dataDir := paths.DaemonDataDir

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "state.db")

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Info("SQLite initialized", slog.String("path", dbPath))
	return db, nil
}

// createTables creates all required tables for the daemon.
func createTables(db *sql.DB) error {
	queries := []string{
		// File positions table for crash recovery
		`CREATE TABLE IF NOT EXISTS file_positions (
			path TEXT PRIMARY KEY,
			offset INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute table creation query: %w", err)
		}
	}

	return nil
}
