// Package token issues and verifies the session JWT. Startup aborts on an empty JWT_SECRET.
package token

import (
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Access JWTs can't be revoked — this is the leak window. Tighten via ACCESS_TOKEN_TTL.
const defaultAccessTTL = 8 * time.Hour

var (
	accessTTLOnce sync.Once
	accessTTL     time.Duration
)

func AccessTokenDuration() time.Duration {
	accessTTLOnce.Do(func() {
		accessTTL = parseDurationEnv("ACCESS_TOKEN_TTL", defaultAccessTTL)
	})
	return accessTTL
}

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

// HS256 keys shorter than 32 bytes are brute-forceable offline (RFC 7518 §3.2).
const MinSecretBytes = 32

// Email is informational — UserID is the canonical reference.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var (
	jwtSecretOnce sync.Once
	jwtSecret     []byte
)

// Aborts on a missing or short secret — an empty one would accept forged tokens.
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

// No audience check here — Parse rejects WS tickets, ParseWSTicket requires them.
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

// Rejects WS tickets: one leaked from a URL must not open the REST API.
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

// SameSite=None needs Secure; the stricter default is kept otherwise.
func AuthCookieSameSite(dflt http.SameSite, production, crossSite bool) (http.SameSite, bool) {
	if crossSite {
		return http.SameSiteNoneMode, true
	}
	return dflt, production
}

// Secure only in production, so local HTTP still works.
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

const wsTicketAudience = "ws"

// Short: the ticket travels in the URL and lands in access logs.
const defaultWSTicketTTL = 30 * time.Second

var (
	wsTicketTTLOnce sync.Once
	wsTicketTTL     time.Duration
)

func WSTicketDuration() time.Duration {
	wsTicketTTLOnce.Do(func() {
		wsTicketTTL = parseDurationEnv("WS_TICKET_TTL", defaultWSTicketTTL)
	})
	return wsTicketTTL
}

// docs/adr/0005 explains why the handshake can't use the cookie.
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

// Rejects session tokens — this path must not accept a long-lived credential.
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
