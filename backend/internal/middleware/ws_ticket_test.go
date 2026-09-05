package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain seeds JWT_SECRET before any test signs or parses a token. secret()
// calls os.Exit on an empty value, which would kill the whole test binary
// instead of failing one case. Must be >= token.MinSecretBytes.
func TestMain(m *testing.M) {
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-secret-do-not-use-in-prod-0123456789")
	}
	os.Exit(m.Run())
}

// serveWSTicket runs RequireWSTicket over a request carrying the given raw
// ticket value ("" omits the query param), reporting whether the wrapped
// handler ran and which user ID reached it.
func serveWSTicket(rawTicket string) (rec *httptest.ResponseRecorder, called bool, gotUserID string) {
	url := "/ws/" + testBoardID
	if rawTicket != "" {
		url += "?ticket=" + rawTicket
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotUserID, _ = r.Context().Value(UserIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	rec = httptest.NewRecorder()
	RequireWSTicket(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec, called, gotUserID
}

func TestRequireWSTicket_ValidTicket_InjectsUserID(t *testing.T) {
	ticket, err := token.GenerateWSTicket(testUserID)
	require.NoError(t, err)

	rec, called, gotUserID := serveWSTicket(ticket)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
	assert.Equal(t, testUserID, gotUserID)
}

func TestRequireWSTicket_MissingTicket_Returns401(t *testing.T) {
	rec, called, _ := serveWSTicket("")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

func TestRequireWSTicket_MalformedTicket_Returns401(t *testing.T) {
	rec, called, _ := serveWSTicket("not-a-jwt")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

// A session access token must not open the WS handshake — the ticket path is
// deliberately not a second way to present a long-lived credential.
func TestRequireWSTicket_AccessToken_Returns401(t *testing.T) {
	access, err := token.Generate(testUserID, "user@example.com")
	require.NoError(t, err)

	rec, called, _ := serveWSTicket(access)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

// The cookie path must not accept a WS ticket either: RequireAuth reads the
// same secret, so only the audience check keeps the two apart.
func TestRequireAuth_WSTicketAsBearer_Returns401(t *testing.T) {
	ticket, err := token.GenerateWSTicket(testUserID)
	require.NoError(t, err)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req.Header.Set("Authorization", "Bearer "+ticket)
	rec := httptest.NewRecorder()
	RequireAuth(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}
