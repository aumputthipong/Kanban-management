package httputil

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

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

type APIFunc func(w http.ResponseWriter, r *http.Request) error

// 5xx log at Error and go to Sentry; 4xx log at Info.
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
			// The hub is only on the context when SENTRY_DSN is set.
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
