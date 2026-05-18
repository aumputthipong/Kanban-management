package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// sensitivePathPrefixes are request paths whose query string must be redacted
// before being logged. Today only the OAuth callback carries short-lived but
// sensitive material (`code`, `state`) in its query string; extend this list
// rather than adding ad-hoc redactions per call site.
var sensitivePathPrefixes = []string{
	"/api/auth/google/callback",
}

// RequestLogger is a drop-in replacement for chi's stdlib Logger that emits
// structured logs via slog and redacts query strings on sensitive paths.
// chi's default logger prints the full URL including raw query — OAuth
// auth-codes would land in stdout, Sentry breadcrumbs, and any log aggregator
// downstream. Redacting at the logger keeps the rest of the pipeline honest
// without having to trust every sink.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		path := r.URL.Path
		query := r.URL.RawQuery
		if isSensitivePath(path) && query != "" {
			query = "[REDACTED]"
		}

		ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			slog.Info("http request",
				"method", r.Method,
				"path", path,
				"query", query,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"request_id", chiMiddleware.GetReqID(r.Context()),
			)
		}()
		next.ServeHTTP(ww, r)
	})
}

func isSensitivePath(path string) bool {
	for _, p := range sensitivePathPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
