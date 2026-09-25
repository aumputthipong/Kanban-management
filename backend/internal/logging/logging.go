// Package logging installs slog.Default: JSON at WARN in production, text at INFO otherwise.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

func Init() {
	production := os.Getenv("ENV") == "production"
	level := parseLevel(os.Getenv("LOG_LEVEL"), production)

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if production {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func parseLevel(raw string, production bool) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	if production {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}
