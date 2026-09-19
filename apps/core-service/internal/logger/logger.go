// Package logger builds the single *slog.Logger the rest of the service logs
// through, configured from config.Config (LOG_LEVEL / LOG_FORMAT) instead of
// each package reaching for the stdlib "log" package with its own ad-hoc
// format.
package logger

import (
	"log/slog"
	"os"
	"strings"

	config "github.com/DewaSRY/core-service/internal/config"
)

func New(cfg config.Config) *slog.Logger {
	level := parseLevel(cfg.LogLevel)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level <= slog.LevelDebug,
	}

	var handler slog.Handler
	switch {
	case strings.EqualFold(cfg.LogFormat, "text"):
		handler = slog.NewTextHandler(os.Stdout, opts)
	case cfg.LogPrettyJSON:
		handler = newPrettyHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
