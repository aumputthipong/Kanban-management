package httputil

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// APIError is a typed error that carries an HTTP status code and a user-facing message.
type APIError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func NewAPIError(statusCode int, message string, err error) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// APIFunc is an http.HandlerFunc variant that returns an error.
type APIFunc func(w http.ResponseWriter, r *http.Request) error

// MakeHandler wraps an APIFunc, centralising error handling, logging, and error
// reporting. Every handler error funnels through here, so this is the single
// place that ties a failure back to its request:
//
//   - logs via slog (structured, matches the rest of the pipeline) with the
//     request_id, so an error line can be correlated with the access-log line
//     and the Sentry event for the same request;
//   - 5xx are logged at Error and reported to Sentry (panics are already
//     captured by SentryRecoverer; this covers the far more common handled-500
//     path — DB errors and the like). 4xx are client-driven, logged at Info and
//     not sent to Sentry.
func MakeHandler(h APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		status := http.StatusInternalServerError
		message := "Internal server error"
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			status = apiErr.StatusCode
			message = apiErr.Message
		}

		reqID := chiMiddleware.GetReqID(r.Context())

		if status >= http.StatusInternalServerError {
			slog.Error("request failed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"request_id", reqID,
				"error", err.Error(),
			)
			// Report server errors when Sentry is configured. The hub is only
			// on the context when SentryRecoverer mounted it (SENTRY_DSN set);
			// nil hub => Sentry disabled, skip silently.
			if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
				hub.CaptureException(err)
			}
		} else {
			slog.Info("request rejected",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"request_id", reqID,
				"error", err.Error(),
			)
		}

		RespondError(w, status, message)
	}
}
