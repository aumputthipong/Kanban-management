package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
	"github.com/stretchr/testify/assert"
)

// TestMain seeds JWT_SECRET so issueSession can sign tokens. The handler's
// token.Generate fatals on empty secret — this would crash the test binary
// rather than fail a single test, so we set it before any test runs. Must be
// ≥32 bytes: the security hardening rejects shorter secrets at startup.
func TestMain(m *testing.M) {
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-secret-do-not-use-in-prod-0123456789")
	}
	os.Exit(m.Run())
}

// findCookie returns the value of the named Set-Cookie header from a
// recorded response, or "" if the cookie was not set.
func findCookie(w *httptest.ResponseRecorder, name string) (value string, found bool) {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c.Value, true
		}
	}
	return "", false
}

func newTestAuthHandler(svc *mock.MockAuthService) *AuthHandler {
	// production=false → cookies are not Secure-only, so httptest captures them.
	return NewAuthHandler(svc, false, false)
}

// ────────────────────────────────────────────────
// Register
// ────────────────────────────────────────────────

func TestRegister_Success_SetsAuthCookies(t *testing.T) {
	svc := &mock.MockAuthService{
		RegisterFn: func(ctx context.Context, arg service.RegisterParams) (db.User, error) {
			assert.Equal(t, "jane@example.com", arg.Email)
			return db.User{ID: validUserID, Email: arg.Email, FullName: arg.FullName}, nil
		},
		IssueRefreshTokenFn: func(ctx context.Context, userID, ua, ip string) (string, error) {
			return "raw-refresh-token", nil
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"jane@example.com","full_name":"Jane Doe","password":"hunter22!"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Register)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	_, hasAuth := findCookie(w, "auth_token")
	assert.True(t, hasAuth, "Register must set the access-token cookie")
	refresh, hasRefresh := findCookie(w, token.RefreshCookieName)
	assert.True(t, hasRefresh, "Register must set the refresh-token cookie")
	assert.Equal(t, "raw-refresh-token", refresh, "refresh cookie should hold the raw token, not a hash")
}

func TestRegister_EmailTaken_Returns409(t *testing.T) {
	svc := &mock.MockAuthService{
		RegisterFn: func(ctx context.Context, arg service.RegisterParams) (db.User, error) {
			return db.User{}, service.ErrEmailTaken
		},
		IssueRefreshTokenFn: func(ctx context.Context, userID, ua, ip string) (string, error) {
			t.Fatal("must not issue refresh token when registration failed")
			return "", nil
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"taken@example.com","full_name":"Jane","password":"hunter22!"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Register)(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	_, hasAuth := findCookie(w, "auth_token")
	assert.False(t, hasAuth, "no auth cookie on failed register")
}

func TestRegister_ShortPassword_Returns400(t *testing.T) {
	svc := &mock.MockAuthService{
		RegisterFn: func(ctx context.Context, arg service.RegisterParams) (db.User, error) {
			t.Fatal("must not call service when validation fails")
			return db.User{}, nil
		},
	}
	h := newTestAuthHandler(svc)

	// password "short" is 5 chars — fails min=8 validator. The validator must
	// run before the service so we never persist a weak password.
	body := strings.NewReader(`{"email":"x@y.com","full_name":"X","password":"short"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Register)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_InvalidEmail_Returns400(t *testing.T) {
	svc := &mock.MockAuthService{}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"not-an-email","full_name":"X","password":"hunter22!"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Register)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockAuthService{
		RegisterFn: func(ctx context.Context, arg service.RegisterParams) (db.User, error) {
			return db.User{}, errors.New("db down")
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"x@y.com","full_name":"X","password":"hunter22!"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Register)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// Login
// ────────────────────────────────────────────────

func TestLogin_Success_SetsCookies(t *testing.T) {
	svc := &mock.MockAuthService{
		LoginFn: func(ctx context.Context, email, password string) (db.User, error) {
			return db.User{ID: validUserID, Email: email, FullName: "Jane"}, nil
		},
		IssueRefreshTokenFn: func(ctx context.Context, userID, ua, ip string) (string, error) {
			return "fresh-refresh", nil
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"jane@example.com","password":"hunter22!"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Login)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_, hasAuth := findCookie(w, "auth_token")
	assert.True(t, hasAuth)
	_, hasRefresh := findCookie(w, token.RefreshCookieName)
	assert.True(t, hasRefresh)
}

// TestLogin_InvalidCreds_Returns401_NoDetail — the handler collapses
// ErrInvalidCreds and ErrOAuthOnly into the same "Invalid credentials"
// response. This is intentional: distinguishing them would let an attacker
// learn "this email exists but is OAuth-only" → user enumeration.
func TestLogin_InvalidCreds_Returns401_NoDetail(t *testing.T) {
	svc := &mock.MockAuthService{
		LoginFn: func(ctx context.Context, email, password string) (db.User, error) {
			return db.User{}, service.ErrInvalidCreds
		},
		IssueRefreshTokenFn: func(ctx context.Context, userID, ua, ip string) (string, error) {
			t.Fatal("must not issue tokens on failed login")
			return "", nil
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"jane@example.com","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Login)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
}

// TestLogin_OAuthOnly_Returns401_SameMessage — guards the anti-enumeration
// behavior described above. If someone "improves" the UX by surfacing
// "this account uses Google OAuth", this test must fail loudly.
func TestLogin_OAuthOnly_Returns401_SameMessage(t *testing.T) {
	svc := &mock.MockAuthService{
		LoginFn: func(ctx context.Context, email, password string) (db.User, error) {
			return db.User{}, service.ErrOAuthOnly
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"jane@example.com","password":"anything"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Login)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
	assert.NotContains(t, strings.ToLower(w.Body.String()), "oauth", "must not leak provider hint")
	assert.NotContains(t, strings.ToLower(w.Body.String()), "google", "must not leak provider hint")
}

func TestLogin_MissingPassword_Returns400(t *testing.T) {
	svc := &mock.MockAuthService{}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"x@y.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Login)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockAuthService{
		LoginFn: func(ctx context.Context, email, password string) (db.User, error) {
			return db.User{}, errors.New("db down")
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"x@y.com","password":"hunter22"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Login)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// OAuthCallback
// ────────────────────────────────────────────────

func TestOAuthCallback_Success(t *testing.T) {
	svc := &mock.MockAuthService{
		UpsertOAuthUserFn: func(ctx context.Context, email, fullName, provider, providerID string) (db.User, error) {
			assert.Equal(t, "google", provider)
			return db.User{ID: validUserID, Email: email, FullName: fullName}, nil
		},
		IssueRefreshTokenFn: func(ctx context.Context, userID, ua, ip string) (string, error) {
			return "raw", nil
		},
	}
	h := newTestAuthHandler(svc)

	body := strings.NewReader(`{"email":"jane@example.com","full_name":"Jane","provider":"google","provider_id":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.OAuthCallback)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_, hasAuth := findCookie(w, "auth_token")
	assert.True(t, hasAuth)
}

func TestOAuthCallback_UnknownProvider_Returns400(t *testing.T) {
	svc := &mock.MockAuthService{
		UpsertOAuthUserFn: func(ctx context.Context, email, fullName, provider, providerID string) (db.User, error) {
			t.Fatal("must reject unknown provider before service call")
			return db.User{}, nil
		},
	}
	h := newTestAuthHandler(svc)

	// validator: provider must be oneof=google github.
	body := strings.NewReader(`{"email":"x@y.com","full_name":"X","provider":"facebook","provider_id":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth", body)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.OAuthCallback)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ────────────────────────────────────────────────
// Logout
// ────────────────────────────────────────────────

func TestLogout_WithRefreshCookie_RevokesAndClearsBoth(t *testing.T) {
	revokedTokens := []string{}
	svc := &mock.MockAuthService{
		RevokeRefreshTokenFn: func(ctx context.Context, rawToken string) error {
			revokedTokens = append(revokedTokens, rawToken)
			return nil
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: token.RefreshCookieName, Value: "raw-token-to-revoke"})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Logout)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{"raw-token-to-revoke"}, revokedTokens, "server-side revoke must run so token can't be reused")

	// Both cookies should be expired (MaxAge<0 or empty value).
	authCookie, _ := findCookie(w, "auth_token")
	assert.Equal(t, "", authCookie, "auth cookie should be cleared on logout")
}

// TestLogout_NoCookie_StillReturns204 — logout is idempotent. A client with
// no cookies (already logged out / never logged in) must not receive an
// error, otherwise the UI would loop on a 4xx during double-logout.
func TestLogout_NoCookie_StillReturns204(t *testing.T) {
	svc := &mock.MockAuthService{
		RevokeRefreshTokenFn: func(ctx context.Context, rawToken string) error {
			t.Fatal("must not call revoke when no refresh cookie is present")
			return nil
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Logout)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestLogout_RevokeFails_StillReturns204 — by design Logout swallows the
// revoke error (logged via slog) so a transient DB blip doesn't trap the
// user in a logged-in state on the client.
func TestLogout_RevokeFails_StillReturns204(t *testing.T) {
	svc := &mock.MockAuthService{
		RevokeRefreshTokenFn: func(ctx context.Context, rawToken string) error {
			return errors.New("db down")
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: token.RefreshCookieName, Value: "raw"})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Logout)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ────────────────────────────────────────────────
// Refresh
// ────────────────────────────────────────────────

func TestRefresh_Success_RotatesBothCookies(t *testing.T) {
	svc := &mock.MockAuthService{
		RotateRefreshTokenFn: func(ctx context.Context, raw, ua, ip string) (service.RefreshRotationResult, error) {
			assert.Equal(t, "old-raw", raw)
			return service.RefreshRotationResult{
				UserID:    validUserID,
				UserEmail: "jane@example.com",
				RawToken:  "new-raw",
			}, nil
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: token.RefreshCookieName, Value: "old-raw"})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Refresh)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	_, hasAuth := findCookie(w, "auth_token")
	assert.True(t, hasAuth, "must issue new access cookie")
	refresh, hasRefresh := findCookie(w, token.RefreshCookieName)
	assert.True(t, hasRefresh, "must rotate refresh cookie")
	assert.Equal(t, "new-raw", refresh, "rotated cookie must carry the new raw token, not the old one")
}

func TestRefresh_NoCookie_Returns401(t *testing.T) {
	svc := &mock.MockAuthService{
		RotateRefreshTokenFn: func(ctx context.Context, raw, ua, ip string) (service.RefreshRotationResult, error) {
			t.Fatal("must not rotate when no cookie is present")
			return service.RefreshRotationResult{}, nil
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Refresh)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRefresh_InvalidToken_Returns401AndClearsCookie — any rotation failure
// returns 401 (intentionally undifferentiated to prevent probing for valid-
// but-expired tokens) and clears the cookie so the browser stops sending it.
func TestRefresh_InvalidToken_Returns401AndClearsCookie(t *testing.T) {
	svc := &mock.MockAuthService{
		RotateRefreshTokenFn: func(ctx context.Context, raw, ua, ip string) (service.RefreshRotationResult, error) {
			return service.RefreshRotationResult{}, errors.New("token expired")
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: token.RefreshCookieName, Value: "bad-or-expired"})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Refresh)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// ClearRefreshCookie sets MaxAge<0 (or expires in the past) — captured
	// here as a present-but-empty cookie value.
	for _, c := range w.Result().Cookies() {
		if c.Name == token.RefreshCookieName {
			assert.True(t, c.MaxAge < 0 || c.Value == "", "refresh cookie must be cleared on failure")
			return
		}
	}
	t.Fatal("expected refresh cookie clear directive in response")
}

// ────────────────────────────────────────────────
// Me
// ────────────────────────────────────────────────

func TestMe_Success(t *testing.T) {
	svc := &mock.MockAuthService{
		GetUserByIDFn: func(ctx context.Context, userID string) (db.GetUserByIDRow, error) {
			assert.Equal(t, validUserID, userID)
			return db.GetUserByIDRow{ID: userID, Email: "jane@example.com", FullName: "Jane"}, nil
		},
	}
	h := newTestAuthHandler(svc)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/auth/me", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Me)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "jane@example.com")
}

func TestMe_MissingUserID_Returns401(t *testing.T) {
	svc := &mock.MockAuthService{
		GetUserByIDFn: func(ctx context.Context, userID string) (db.GetUserByIDRow, error) {
			t.Fatal("must not query user when auth context is missing")
			return db.GetUserByIDRow{}, nil
		},
	}
	h := newTestAuthHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil) // no userID in context
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Me)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockAuthService{
		GetUserByIDFn: func(ctx context.Context, userID string) (db.GetUserByIDRow, error) {
			return db.GetUserByIDRow{}, errors.New("db down")
		},
	}
	h := newTestAuthHandler(svc)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/auth/me", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.Me)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
