package token

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain seeds JWT_SECRET before any test signs or parses a token. secret()
// calls os.Exit on an empty value, which would kill the whole test binary
// instead of failing one case. Must be >= MinSecretBytes.
func TestMain(m *testing.M) {
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-secret-do-not-use-in-prod-0123456789")
	}
	os.Exit(m.Run())
}

func TestParseWSTicket_FreshTicket_ReturnsUserID(t *testing.T) {
	ticket, err := GenerateWSTicket("user-1")
	require.NoError(t, err)

	claims, err := ParseWSTicket(ticket)

	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
}

// The whole point of the `aud` claim: a ticket lifted out of a URL or an
// access log must not open the REST API.
func TestParse_WSTicket_Rejected(t *testing.T) {
	ticket, err := GenerateWSTicket("user-1")
	require.NoError(t, err)

	_, err = Parse(ticket)

	assert.Error(t, err)
}

// The mirror case: the WS handshake must not become a second place to present
// a long-lived session credential.
func TestParseWSTicket_AccessToken_Rejected(t *testing.T) {
	access, err := Generate("user-1", "user@example.com")
	require.NoError(t, err)

	_, err = ParseWSTicket(access)

	assert.Error(t, err)
}

func TestParse_AccessToken_StillAccepted(t *testing.T) {
	access, err := Generate("user-1", "user@example.com")
	require.NoError(t, err)

	claims, err := Parse(access)

	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
}

func TestParseWSTicket_ExpiredTicket_Rejected(t *testing.T) {
	claims := Claims{
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{wsTicketAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
		},
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
	require.NoError(t, err)

	_, err = ParseWSTicket(expired)

	assert.Error(t, err)
}

func TestParseWSTicket_ForeignSignature_Rejected(t *testing.T) {
	claims := Claims{
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{wsTicketAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte("some-other-secret-that-is-long-enough-0123"))
	require.NoError(t, err)

	_, err = ParseWSTicket(forged)

	assert.Error(t, err)
}
