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

// Rotation slides an active session forward, so only idle sessions expire (docs/adr/0001).
const defaultRefreshTTL = 30 * 24 * time.Hour

var (
	refreshTTLOnce sync.Once
	refreshTTL     time.Duration
)

func RefreshTokenDuration() time.Duration {
	refreshTTLOnce.Do(func() {
		refreshTTL = parseDurationEnv("REFRESH_TOKEN_TTL", defaultRefreshTTL)
	})
	return refreshTTL
}

const RefreshCookieName = "refresh_token"

// Only sent to the refresh endpoint.
const RefreshCookiePath = "/api/auth/refresh"

const refreshTokenBytes = 32

// Opaque, not a JWT — docs/adr/0001.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Only the hash is stored.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

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

// Path must match the one used at write time, or browsers won't clear it.
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
