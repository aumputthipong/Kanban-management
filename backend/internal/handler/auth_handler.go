// internal/handler/auth_handler.go
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
)

type AuthHandler struct {
	authService service.AuthServicer
	production  bool
	crossSite   bool
}

func NewAuthHandler(authService service.AuthServicer, production, crossSite bool) *AuthHandler {
	return &AuthHandler{authService: authService, production: production, crossSite: crossSite}
}

// issueSession signs the short-lived access JWT and provisions a fresh refresh
// token, setting both cookies on w. It is shared by Register, Login, and the
// programmatic OAuth callback. Refresh-token issuance failure is logged but
// not fatal — the user still gets a (short) access token; they will simply
// have to re-login when it expires.
func (h *AuthHandler) issueSession(w http.ResponseWriter, r *http.Request, userID, email string) error {
	accessTok, err := token.Generate(userID, email)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to generate token", err)
	}
	token.SetAuthCookie(w, accessTok, h.production, h.crossSite)

	refreshRaw, err := h.authService.IssueRefreshToken(r.Context(), userID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		slog.Error("issue refresh token", "user_id", userID, "err", err)
		return nil
	}
	token.SetRefreshCookie(w, refreshRaw, h.production, h.crossSite)
	return nil
}

// authUserResponse is the shared response body for register / login / oauth.
type authUserResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type registerRequest struct {
	Email    string `json:"email"     validate:"required,email"`
	FullName string `json:"full_name" validate:"required,min=1,max=120"`
	Password string `json:"password"  validate:"required,min=8,max=128"`
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type oauthRequest struct {
	Email      string `json:"email"       validate:"required,email"`
	FullName   string `json:"full_name"   validate:"required,min=1,max=120"`
	Provider   string `json:"provider"    validate:"required,oneof=google github"`
	ProviderID string `json:"provider_id" validate:"required"`
}

// Register creates a new credentials user and sets an auth cookie.
//
// @Summary  Register
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    payload body     registerRequest    true  "Email, full name, password (min 8)"
// @Success  201     {object} authUserResponse
// @Failure  400     {object} httputil.ErrorResponse "validation error"
// @Failure  409     {object} httputil.ErrorResponse "email already in use"
// @Router   /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req registerRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	user, err := h.authService.Register(r.Context(), service.RegisterParams{
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			return httputil.NewAPIError(http.StatusConflict, "Email already in use", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to register", err)
	}

	if err := h.issueSession(w, r, user.ID, user.Email); err != nil {
		return err
	}
	httputil.RespondJSON(w, http.StatusCreated, authUserResponse{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
	})
	return nil
}

// Login authenticates with email + password and sets an auth cookie.
//
// @Summary  Login
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    payload body     loginRequest        true  "Credentials"
// @Success  200     {object} authUserResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  401     {object} httputil.ErrorResponse  "invalid credentials"
// @Router   /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req loginRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	user, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) || errors.Is(err, service.ErrOAuthOnly) {
			return httputil.NewAPIError(http.StatusUnauthorized, "Invalid credentials", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to login", err)
	}

	if err := h.issueSession(w, r, user.ID, user.Email); err != nil {
		return err
	}
	httputil.RespondJSON(w, http.StatusOK, authUserResponse{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
	})
	return nil
}

// OAuthCallback upserts the user from a verified OAuth provider payload and
// sets an auth cookie. Called by the frontend after NextAuth completes the
// provider handshake.
//
// @Summary  OAuth callback (programmatic)
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    payload body     oauthRequest       true  "Verified provider payload"
// @Success  200     {object} authUserResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Router   /api/auth/oauth [post]
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) error {
	var req oauthRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	user, err := h.authService.UpsertOAuthUser(r.Context(), req.Email, req.FullName, req.Provider, req.ProviderID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to authenticate", err)
	}

	if err := h.issueSession(w, r, user.ID, user.Email); err != nil {
		return err
	}
	httputil.RespondJSON(w, http.StatusOK, authUserResponse{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
	})
	return nil
}

// Logout clears the auth cookie and revokes the refresh token server-side so
// it cannot be reused even if the cookie was captured. Always returns 204 —
// missing or already-revoked tokens are not an error for the client.
//
// @Summary  Logout
// @Tags     auth
// @Success  204
// @Router   /api/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie(token.RefreshCookieName); err == nil {
		if revokeErr := h.authService.RevokeRefreshToken(r.Context(), c.Value); revokeErr != nil {
			slog.Error("revoke refresh on logout", "err", revokeErr)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	token.ClearRefreshCookie(w, h.production, h.crossSite)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Refresh rotates the refresh token and issues a new access token. The
// presented refresh cookie is invalidated whether or not the call succeeds —
// rotation is the only mode, there is no "use the same token twice" path.
// 401 on any failure; the frontend interceptor reads that as "session over"
// and redirects to /login.
//
// @Summary  Refresh session
// @Tags     auth
// @Produce  json
// @Success  204
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie(token.RefreshCookieName)
	if err != nil {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	result, err := h.authService.RotateRefreshToken(r.Context(), c.Value, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		// Both invalid and expired surface as 401 to the client; differentiating
		// them would let an attacker probe for valid-but-expired tokens. Clear
		// the cookie either way so the browser stops sending it.
		token.ClearRefreshCookie(w, h.production, h.crossSite)
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	accessTok, err := token.Generate(result.UserID, result.UserEmail)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to generate token", err)
	}
	token.SetAuthCookie(w, accessTok, h.production, h.crossSite)
	token.SetRefreshCookie(w, result.RawToken, h.production, h.crossSite)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Me returns the authenticated user's profile.
//
// @Summary  Current user
// @Tags     auth
// @Produce  json
// @Security CookieAuth
// @Success  200 {object} authUserResponse
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load user", err)
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]string{
		"user_id":   user.ID,
		"email":     user.Email,
		"full_name": user.FullName,
	})
	return nil
}

// wsTicketResponse is the body of GET /api/ws-ticket.
type wsTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
}

// WSTicket issues a short-lived, WebSocket-only token for the handshake.
//
// @Summary  Issue a WebSocket auth ticket
// @Tags     auth
// @Produce  json
// @Security CookieAuth
// @Success  200 {object} wsTicketResponse
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/ws-ticket [get]
func (h *AuthHandler) WSTicket(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	ticket, err := token.GenerateWSTicket(userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to issue ticket", err)
	}

	httputil.RespondJSON(w, http.StatusOK, wsTicketResponse{
		Ticket:    ticket,
		ExpiresIn: int(token.WSTicketDuration().Seconds()),
	})
	return nil
}
