// Package token issues and verifies the session-auth JWT and sets the auth_token
// HttpOnly cookie. The signing secret is read once from JWT_SECRET; the process aborts
// on startup if it is empty.
package token

import (
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// defaultAccessTTL is the exposure window for a leaked access JWT (they cannot be
// revoked) and also how long an idle tab survives a reload before bouncing to /login.
// 8h keeps a work session alive; tighten via ACCESS_TOKEN_TTL and lean on refresh.
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
		slog.Warn("token: invalid duration env, using default", "key", key, "value", v, "default", def.String())
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

// secret returns the JWT signing key, initialising it on first use. It aborts the
// process when JWT_SECRET is missing or too short — running with an empty secret would
// silently accept forged tokens.
func secret() []byte {
	jwtSecretOnce.Do(func() {
		s := os.Getenv("JWT_SECRET")
		if s == "" {
			slog.Error("JWT_SECRET is required")
			os.Exit(1)
		}
		if len(s) < MinSecretBytes {
			slog.Error("JWT_SECRET too short — generate one with: openssl rand -base64 32",
				"min_bytes", MinSecretBytes, "got_bytes", len(s))
			os.Exit(1)
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

// parseSigned verifies the signature and standard claims without inspecting
// the audience. Callers pick the audience rule: Parse rejects WS tickets,
// ParseWSTicket requires one.
func parseSigned(tokenStr string) (*Claims, error) {
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

// Parse validates a session access token, rejecting non-HMAC signing and WS tickets — a
// ticket leaked from a URL must not open the REST API. Any non-nil error means
// "unauthenticated"; it never says which check failed.
func Parse(tokenStr string) (*Claims, error) {
	claims, err := parseSigned(tokenStr)
	if err != nil {
		return nil, err
	}
	if hasAudience(claims, wsTicketAudience) {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// AuthCookieSameSite returns the SameSite mode and Secure flag for an auth cookie. A
// cross-site deploy needs SameSite=None, which browsers only accept with Secure; the
// stricter default is kept otherwise. dflt is the cookie's own default.
func AuthCookieSameSite(dflt http.SameSite, production, crossSite bool) (http.SameSite, bool) {
	if crossSite {
		return http.SameSiteNoneMode, true
	}
	return dflt, production
}

// SetAuthCookie writes the signed JWT into the `auth_token` HttpOnly cookie.
// The Secure flag is set only in production (passed in from main) so local
// HTTP development is not blocked by the browser refusing to send the cookie.
func SetAuthCookie(w http.ResponseWriter, tokenStr string, production, crossSite bool) {
	sameSite, secure := AuthCookieSameSite(http.SameSiteLaxMode, production, crossSite)
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int(AccessTokenDuration().Seconds()),
	})
}

// wsTicketAudience is the `aud` claim that marks a token as usable only for
// the WebSocket handshake. Parse rejects it; ParseWSTicket requires it.
const wsTicketAudience = "ws"

// defaultWSTicketTTL is the lifetime of a WS ticket. It travels in the URL and so lands
// in upstream access logs we do not control — keep the replay window short; it only has
// to survive one handshake.
const defaultWSTicketTTL = 30 * time.Second

var (
	wsTicketTTLOnce sync.Once
	wsTicketTTL     time.Duration
)

// WSTicketDuration returns the WS-ticket lifetime, read once from
// WS_TICKET_TTL (a Go duration string such as "30s").
func WSTicketDuration() time.Duration {
	wsTicketTTLOnce.Do(func() {
		wsTicketTTL = parseDurationEnv("WS_TICKET_TTL", defaultWSTicketTTL)
	})
	return wsTicketTTL
}

// GenerateWSTicket signs a short-lived token that authenticates one WebSocket
// handshake. See docs/adr/0005-websocket-ticket-auth.md for why the handshake
// cannot use the auth cookie.
func GenerateWSTicket(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{wsTicketAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(WSTicketDuration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret())
}

// ParseWSTicket validates a WS ticket and returns its claims. A session access
// token is rejected: the ticket path is deliberately not a second way to
// present a long-lived credential.
func ParseWSTicket(tokenStr string) (*Claims, error) {
	claims, err := parseSigned(tokenStr)
	if err != nil {
		return nil, err
	}
	if !hasAudience(claims, wsTicketAudience) {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func hasAudience(claims *Claims, want string) bool {
	for _, aud := range claims.Audience {
		if aud == want {
			return true
		}
	}
	return false
}
