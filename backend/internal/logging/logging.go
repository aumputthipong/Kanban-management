// Package logging configures the process-wide structured logger: JSON at WARN in
// production, human-readable text at INFO otherwise, overridable with LOG_LEVEL. It is
// installed as slog.Default() so no package needs a *slog.Logger dependency.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures slog.Default from the environment. Safe to call once at boot.
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
