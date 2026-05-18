// Package token issues and verifies the JWT used for session auth, plus the
// helper that sets the `auth_token` HttpOnly cookie. The signing secret is
// loaded once from the JWT_SECRET environment variable; the process aborts
// on startup if it is empty.
package token

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// defaultAccessTTL is how long an issued access token stays valid when
// ACCESS_TOKEN_TTL is not set. A leaked access JWT cannot be revoked, so this
// is the exposure window — but it is also the window after which a plain page
// reload (Next middleware reads the auth_token cookie) bounces an idle user to
// /login, because the refresh token is scoped to the refresh endpoint and is
// not visible to server-side navigation. 8h keeps a normal work session alive
// without a relogin while still expiring same-day. Tighten in production via
// ACCESS_TOKEN_TTL and lean on the rotating refresh token for renewal.
const defaultAccessTTL = 8 * time.Hour

var (
	accessTTLOnce sync.Once
	accessTTL     time.Duration
)

// AccessTokenDuration returns the access-token lifetime, read once from
// ACCESS_TOKEN_TTL (a Go duration string such as "15m", "8h"). The cookie
// MaxAge mirrors this value.
func AccessTokenDuration() time.Duration {
	accessTTLOnce.Do(func() {
		accessTTL = parseDurationEnv("ACCESS_TOKEN_TTL", defaultAccessTTL)
	})
	return accessTTL
}

// parseDurationEnv reads a Go-duration env var, falling back to def when unset,
// unparseable, or non-positive. Shared by the access + refresh TTL loaders.
func parseDurationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		log.Printf("token: invalid %s=%q, using default %s", key, v, def)
		return def
	}
	return d
}

// MinSecretBytes is the floor enforced on JWT_SECRET. HS256 keys shorter than
// the hash size (32 bytes) are brute-forceable offline once any token is
// captured — RFC 7518 §3.2 says the key SHOULD be the same size as the
// output of the HMAC function (SHA-256 → 32 bytes).
const MinSecretBytes = 32

// Claims is the JWT body for an authenticated user. UserID is the canonical
// reference; Email is included for ergonomics in logs and is not authoritative
// (a user can change their email; the UserID does not).
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var (
	jwtSecretOnce sync.Once
	jwtSecret     []byte
)

// secret returns the JWT signing key, initialising it on first use.
// It log.Fatal's if JWT_SECRET is missing — this is intentional because
// running with an empty secret would silently accept forged tokens.
func secret() []byte {
	jwtSecretOnce.Do(func() {
		s := os.Getenv("JWT_SECRET")
		if s == "" {
			log.Fatal("JWT_SECRET is required")
		}
		if len(s) < MinSecretBytes {
			log.Fatalf("JWT_SECRET must be at least %d bytes (got %d) — generate one with: openssl rand -base64 32", MinSecretBytes, len(s))
		}
		jwtSecret = []byte(s)
	})
	return jwtSecret
}

// Generate signs a new JWT for the given user. The returned string is what
// gets placed in the `auth_token` cookie via SetAuthCookie.
func Generate(userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret())
}

// Parse validates the signed token string and returns its claims. It rejects
// any signing method other than HMAC. Returns jwt.ErrTokenInvalidClaims for
// expired, malformed, or wrong-algorithm tokens — callers should treat any
// non-nil error as "unauthenticated" without leaking which check failed.
func Parse(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret(), nil
	})
	if err != nil || !t.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	claims, ok := t.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// SetAuthCookie writes the signed JWT into the `auth_token` HttpOnly cookie.
// The Secure flag is set only in production (passed in from main) so local
// HTTP development is not blocked by the browser refusing to send the cookie.
func SetAuthCookie(w http.ResponseWriter, tokenStr string, production bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   production,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(AccessTokenDuration().Seconds()),
	})
}
