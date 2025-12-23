package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
)

// InitLogger creates a configured slog.Logger instance.
func InitLogger(cfg *config.Config) *slog.Logger {
	level := parseLogLevel(cfg.Logging.Level)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(cfg.Logging.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
