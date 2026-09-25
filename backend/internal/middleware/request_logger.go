package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// Paths whose query string is redacted (OAuth code/state, WS ticket). Extend here, not per call site.
var sensitivePathPrefixes = []string{
	"/api/auth/google/callback",
	"/ws/",
}

// chi's Logger prints the raw query — OAuth codes and WS tickets would leak.
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
				"client_ip", chiMiddleware.GetClientIP(r.Context()),
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
