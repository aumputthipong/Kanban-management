package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// AuthRateLimit caps brute-force attempts on /api/auth/* endpoints by client IP.
// 20 requests per minute is enough headroom for legitimate retry / typo flows
// while making credential stuffing impractical.
func AuthRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitByIP(20, time.Minute) //nolint:staticcheck // behind a proxy this keys on the proxy IP; fix needs a trusted-header decision
}

// GeneralRateLimit applies a wider cap on the rest of the API to absorb
// runaway client loops without being noticeable for normal use.
func GeneralRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitByIP(300, time.Minute) //nolint:staticcheck // same proxy-IP caveat as AuthRateLimit
}
