package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// defaultRefreshTTL is the refresh-token lifetime when REFRESH_TOKEN_TTL is not
// set. Rotation slides an active session forward, so only an idle session
// expires after this window. See docs/adr/0001-opaque-refresh-tokens.md.
const defaultRefreshTTL = 30 * 24 * time.Hour

var (
	refreshTTLOnce sync.Once
	refreshTTL     time.Duration
)

// RefreshTokenDuration returns the refresh-token lifetime, read once from
// REFRESH_TOKEN_TTL (a Go duration string such as "720h"). Both the DB
// expires_at and the cookie MaxAge use this value.
func RefreshTokenDuration() time.Duration {
	refreshTTLOnce.Do(func() {
		refreshTTL = parseDurationEnv("REFRESH_TOKEN_TTL", defaultRefreshTTL)
	})
	return refreshTTL
}

// RefreshCookieName is the cookie that carries the opaque refresh token.
const RefreshCookieName = "refresh_token"

// RefreshCookiePath scopes the refresh cookie so it is only sent to the refresh
// endpoint, not with ordinary API calls.
const RefreshCookiePath = "/api/auth/refresh"

// refreshTokenBytes is the entropy of the raw token: 256 bits.
const refreshTokenBytes = 32

// GenerateRefreshToken returns an opaque base64url-encoded random token.
// See docs/adr/0001-opaque-refresh-tokens.md for why it is not a JWT.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken returns the sha256-hex digest of the raw token. Only this
// hash is stored; the raw token lives only in the client cookie.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// SetRefreshCookie writes the refresh token cookie: HttpOnly, path-scoped to the
// refresh endpoint, and SameSite=Strict (or None on a cross-site deploy).
func SetRefreshCookie(w http.ResponseWriter, raw string, production, crossSite bool) {
	sameSite, secure := AuthCookieSameSite(http.SameSiteStrictMode, production, crossSite)
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    raw,
		Path:     RefreshCookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int(RefreshTokenDuration().Seconds()),
	})
}

// ClearRefreshCookie expires the refresh cookie. MaxAge=-1 plus matching Path
// is required — browsers will not clear a cookie whose Path differs.
func ClearRefreshCookie(w http.ResponseWriter, production, crossSite bool) {
	sameSite, secure := AuthCookieSameSite(http.SameSiteStrictMode, production, crossSite)
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     RefreshCookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})
}
