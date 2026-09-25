// With SENTRY_DSN empty every path is a no-op.
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

// SENTRY_* options are documented in .env.example.
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

func FlushSentry(timeout time.Duration) {
	sentry.Flush(timeout)
}

// Mount before chi's Recoverer so Sentry sees the panic.
func SentryRecoverer() func(http.Handler) http.Handler {
	if os.Getenv("SENTRY_DSN") == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	handler := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})
	return handler.Handle
}
