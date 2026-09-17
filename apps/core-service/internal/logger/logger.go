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

// New builds a *slog.Logger writing to stdout.
//
//   - LOG_LEVEL: "debug" | "info" | "warn" | "error" (case-insensitive).
//     Empty or unrecognized falls back to "info".
//   - LOG_FORMAT: "json" (default) | "text". JSON is what a log aggregator
//     (CloudWatch, Loki, ...) expects; "text" is easier to read against a
//     local terminal.
//
// At "debug" level, records also carry the source file:line they were
// logged from (slog's AddSource), since that's the situation where you're
// actively tracing through a request rather than just watching an aggregate.
func New(cfg config.Config) *slog.Logger {
	level := parseLevel(cfg.LogLevel)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level <= slog.LevelDebug,
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.LogFormat, "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
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
