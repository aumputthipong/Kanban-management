package middleware

import (
	"net"
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// ClientIPResolver decides which address identifies the caller, for every limiter below.
// trustedProxies is how many proxies sit in front of this server (Render or nginx = 1,
// 0 = directly exposed). Set it too low and a client can forge its own key, so
// docs/DEPLOY.md documents verifying the resolved IP once after a deploy.
func ClientIPResolver(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies < 1 {
		return func(next http.Handler) http.Handler { return chiMiddleware.ClientIPFromRemoteAddr(next) }
	}
	return chiMiddleware.ClientIPFromXFFTrustedProxies(trustedProxies)
}

// keyByClientIP buckets a limiter by caller. The RemoteAddr fallback is not cosmetic:
// the XFF middleware fails closed on a short chain, and httprate puts every empty key
// in ONE shared bucket — the exact failure these limits exist to prevent.
func keyByClientIP(r *http.Request) (string, error) {
	ip := chiMiddleware.GetClientIP(r.Context())
	if ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip = host
	}
	return httprate.CanonicalizeIP(ip), nil
}

// AuthRateLimit caps brute-force attempts on /api/auth/* endpoints by client IP.
// 20 requests per minute is enough headroom for legitimate retry / typo flows
// while making credential stuffing impractical.
func AuthRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitBy(20, time.Minute, keyByClientIP)
}

// DemoRateLimit caps POST /api/auth/demo. Each call writes a user and a whole seeded
// board that lives 24h, so an unlimited loop would fill the database. 30 per hour is
// out of reach for someone clicking, and leaves room for an office NAT where a whole
// team shares one address.
func DemoRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitBy(30, time.Hour, keyByClientIP)
}

// GeneralRateLimit applies a wider cap on the rest of the API to absorb
// runaway client loops without being noticeable for normal use.
func GeneralRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitBy(300, time.Minute, keyByClientIP)
}
