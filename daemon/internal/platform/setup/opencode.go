package setup

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"

	_ "modernc.org/sqlite"
)

// InitOpenCodeDB initializes a read-only SQLite connection to the OpenCode database.
// Returns nil if the database file does not exist (OpenCode not installed).
func InitOpenCodeDB(cfg *Config, logger *slog.Logger) *sql.DB {
	dbPath := pathutil.GetOpenCodeDBPath()
	if dbPath == "" {
		return nil
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		logger.Info("OpenCode DB not found, skipping initialization",
			slog.String("path", dbPath),
		)
		return nil
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		logger.Warn("failed to open OpenCode DB",
			slog.String("path", dbPath),
			slog.Any("error", err),
		)
		return nil
	}

	if err := db.Ping(); err != nil {
		db.Close()
		logger.Warn("failed to ping OpenCode DB",
			slog.String("path", dbPath),
			slog.Any("error", err),
		)
		return nil
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		logger.Warn("failed to enable WAL mode on OpenCode DB",
			slog.String("path", dbPath),
			slog.Any("error", err),
		)
		return nil
	}

	logger.Info("OpenCode DB initialized", slog.String("path", dbPath))
	return db
}
