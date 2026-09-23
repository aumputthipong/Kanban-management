package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/httprate"
)

// limitOne wraps a 1-per-minute limiter in the resolver, mirroring how setupRoutes
// stacks them. Each call gets its own limiter, so buckets never leak between tests.
func limitOne(trustedProxies int) http.Handler {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	return ClientIPResolver(trustedProxies)(httprate.LimitBy(1, time.Minute, keyByClientIP)(ok))
}

func send(h http.Handler, remoteAddr, xff string) int {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func TestRateLimit_BehindProxy_DifferentClientsGetOwnBuckets(t *testing.T) {
	h := limitOne(1)

	if got := send(h, "10.0.0.7:1234", "203.0.113.5"); got != http.StatusOK {
		t.Fatalf("first client: got %d, want 200", got)
	}
	if got := send(h, "10.0.0.7:1234", "198.51.100.9"); got != http.StatusOK {
		t.Errorf("second client shares the proxy's address; it must not inherit the first one's bucket: got %d, want 200", got)
	}
	if got := send(h, "10.0.0.7:1234", "203.0.113.5"); got != http.StatusTooManyRequests {
		t.Errorf("first client's second request: got %d, want 429", got)
	}
}

// A client that sends its own X-Forwarded-For must not win a fresh bucket: the proxy
// appends the real address, and only that rightmost entry is trusted.
func TestRateLimit_ForgedXFF_CannotEscapeItsBucket(t *testing.T) {
	h := limitOne(1)

	if got := send(h, "10.0.0.7:1234", "9.9.9.9, 203.0.113.5"); got != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", got)
	}
	if got := send(h, "10.0.0.7:1234", "8.8.8.8, 203.0.113.5"); got != http.StatusTooManyRequests {
		t.Errorf("forged left-hand entry bought a new bucket: got %d, want 429", got)
	}
}

// With no proxy configured the socket address is the key — and it must stay per-caller.
// An empty key would put every caller in one bucket, which is the bug this replaced.
func TestRateLimit_NoProxy_KeysOnRemoteAddr(t *testing.T) {
	h := limitOne(0)

	if got := send(h, "203.0.113.5:1111", ""); got != http.StatusOK {
		t.Fatalf("first caller: got %d, want 200", got)
	}
	if got := send(h, "198.51.100.9:2222", ""); got != http.StatusOK {
		t.Errorf("second caller: got %d, want 200", got)
	}
	if got := send(h, "203.0.113.5:3333", ""); got != http.StatusTooManyRequests {
		t.Errorf("first caller again (new port, same host): got %d, want 429", got)
	}
}

// Configured for a proxy but reached directly: the XFF chain is too short, the resolver
// fails closed, and the fallback must still separate callers rather than share one bucket.
func TestRateLimit_ProxyConfiguredButHeaderMissing_FallsBackPerCaller(t *testing.T) {
	h := limitOne(1)

	if got := send(h, "203.0.113.5:1111", ""); got != http.StatusOK {
		t.Fatalf("first caller: got %d, want 200", got)
	}
	if got := send(h, "198.51.100.9:2222", ""); got != http.StatusOK {
		t.Errorf("second caller landed in the first one's bucket: got %d, want 200", got)
	}
}

func TestRateLimit_IPv6ClientsBucketByPrefix(t *testing.T) {
	h := limitOne(1)

	if got := send(h, "10.0.0.7:1234", "2001:db8:1::1"); got != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", got)
	}
	// Same /64: a client rotating addresses inside its own prefix must not reset the count.
	if got := send(h, "10.0.0.7:1234", "2001:db8:1::99"); got != http.StatusTooManyRequests {
		t.Errorf("address rotation within one /64 bought a new bucket: got %d, want 429", got)
	}
}
