// Package observability wraps Sentry initialisation and the chi recovery middleware
// that ships panics to it. Env-gated: with SENTRY_DSN empty every path is a no-op, so
// local dev and CI need no Sentry project.
package observability

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// InitSentry initialises the global Sentry hub from env and reports whether it was
// configured. SENTRY_DSN empty disables Sentry entirely; the other SENTRY_* knobs are
// documented in .env.example.
func InitSentry(release string) bool {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		slog.Info("sentry disabled (SENTRY_DSN not set)")
		return false
	}

	env := os.Getenv("SENTRY_ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "development"
	}

	if r := os.Getenv("SENTRY_RELEASE"); r != "" {
		release = r
	}

	tracesRate := 0.1
	if v := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); v != "" {
		if parsed, perr := strconv.ParseFloat(v, 64); perr == nil && parsed >= 0 && parsed <= 1 {
			tracesRate = parsed
		}
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          release,
		AttachStacktrace: true,
		TracesSampleRate: tracesRate,
	})
	if err != nil {
		slog.Error("sentry init failed", "err", err)
		return false
	}
	slog.Info("sentry enabled", "environment", env, "release", release, "traces_sample_rate", tracesRate)
	return true
}

// FlushSentry blocks for up to timeout while Sentry ships any pending events.
// Call from main.go's deferred shutdown so SIGTERM doesn't drop the last error.
func FlushSentry(timeout time.Duration) {
	sentry.Flush(timeout)
}

// SentryRecoverer catches panics from downstream handlers and reports them before
// re-raising. Mount it BEFORE chi's Recoverer so Sentry sees the panic and the stdlib
// middleware still writes the 500. With no DSN it is a pass-through.
func SentryRecoverer() func(http.Handler) http.Handler {
	if os.Getenv("SENTRY_DSN") == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	handler := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})
	return handler.Handle
}
