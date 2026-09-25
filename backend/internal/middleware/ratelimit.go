package middleware

import (
	"net"
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// trustedProxies: proxies in front of this server. Too low and a client can forge its key (docs/DEPLOY.md).
func ClientIPResolver(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies < 1 {
		return func(next http.Handler) http.Handler { return chiMiddleware.ClientIPFromRemoteAddr(next) }
	}
	return chiMiddleware.ClientIPFromXFFTrustedProxies(trustedProxies)
}

// The RemoteAddr fallback matters: httprate puts every empty key in ONE shared bucket.
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

// 20/min: room for typos, too slow for credential stuffing.
func AuthRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitBy(20, time.Minute, keyByClientIP)
}

// Each call seeds a whole board, so it gets its own tighter cap.
func DemoRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitBy(30, time.Hour, keyByClientIP)
}

func GeneralRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitBy(300, time.Minute, keyByClientIP)
}
