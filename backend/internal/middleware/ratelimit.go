package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// trustedProxies: proxies in front of this server. Too low and a client can forge its key (docs/DEPLOY.md).
// A shorter chain is a caller that skipped the outer hop (WS/SSR bypass Vercel); trust all of it.
func ClientIPResolver(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies < 1 {
		return func(next http.Handler) http.Handler { return chiMiddleware.ClientIPFromRemoteAddr(next) }
	}
	return func(next http.Handler) http.Handler {
		byHops := make([]http.Handler, trustedProxies+1)
		for hops := 1; hops <= trustedProxies; hops++ {
			byHops[hops] = chiMiddleware.ClientIPFromXFFTrustedProxies(hops)(next)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hops := min(xffLen(r.Header.Values("X-Forwarded-For")), trustedProxies)
			if hops == 0 {
				next.ServeHTTP(w, r)
				return
			}
			byHops[hops].ServeHTTP(w, r)
		})
	}
}

func xffLen(headers []string) int {
	n := 0
	for _, h := range headers {
		for _, entry := range strings.Split(h, ",") {
			if strings.TrimSpace(entry) != "" {
				n++
			}
		}
	}
	return n
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
