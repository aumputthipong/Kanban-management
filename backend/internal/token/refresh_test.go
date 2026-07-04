package token

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateRefreshTokenIsUniqueAndUrlSafe(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		tok, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if len(tok) < 40 {
			t.Errorf("token too short: %q (%d)", tok, len(tok))
		}
		if strings.ContainsAny(tok, "+/=") {
			t.Errorf("token not base64url: %q", tok)
		}
		if _, dup := seen[tok]; dup {
			t.Errorf("duplicate token issued: %q", tok)
		}
		seen[tok] = struct{}{}
	}
}

func TestHashRefreshTokenIsSHA256Hex(t *testing.T) {
	t.Parallel()
	got := HashRefreshToken("hello")
	sum := sha256.Sum256([]byte("hello"))
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Errorf("hash mismatch: got %s want %s", got, want)
	}
	if len(got) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(got))
	}
}

func TestSetRefreshCookieScopedAndStrict(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	SetRefreshCookie(w, "raw-value", true, false)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != RefreshCookieName {
		t.Errorf("name = %q want %q", c.Name, RefreshCookieName)
	}
	if c.Path != RefreshCookiePath {
		t.Errorf("path = %q want %q (refresh cookie must be scoped — see refresh.go)", c.Path, RefreshCookiePath)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly must be set")
	}
	if !c.Secure {
		t.Error("Secure must be set in production mode")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v want Strict", c.SameSite)
	}
	if c.MaxAge != int(RefreshTokenDuration().Seconds()) {
		t.Errorf("MaxAge = %d want %d", c.MaxAge, int(RefreshTokenDuration().Seconds()))
	}
}

func TestSetRefreshCookieCrossSite(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	// Cross-site deploy (frontend and backend on different sites): SameSite must
	// be None so the browser sends the cookie, and Secure is forced regardless
	// of the production flag.
	SetRefreshCookie(w, "raw-value", false, true)
	c := w.Result().Cookies()[0]
	if c.SameSite != http.SameSiteNoneMode {
		t.Errorf("SameSite = %v want None (cross-site)", c.SameSite)
	}
	if !c.Secure {
		t.Error("Secure must be forced true when SameSite=None")
	}
}

func TestClearRefreshCookieMatchesPath(t *testing.T) {
	t.Parallel()
	// MUST set Path to the same value used at write time or the browser
	// won't delete it — easy footgun, hence the explicit test.
	w := httptest.NewRecorder()
	ClearRefreshCookie(w, false, false)
	c := w.Result().Cookies()[0]
	if c.Path != RefreshCookiePath {
		t.Errorf("clear path = %q want %q", c.Path, RefreshCookiePath)
	}
	if c.MaxAge != -1 {
		t.Errorf("MaxAge = %d want -1 (expire now)", c.MaxAge)
	}
}
