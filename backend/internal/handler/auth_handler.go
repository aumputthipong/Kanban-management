package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
)

type AuthHandler struct {
	authService service.AuthServicer
	demoService service.DemoServicer
	production  bool
	crossSite   bool
}

func NewAuthHandler(authService service.AuthServicer, demoService service.DemoServicer, production, crossSite bool) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		demoService: demoService,
		production:  production,
		crossSite:   crossSite,
	}
}

// A refresh-token failure is logged, not fatal: the access token still works until expiry.
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

type demoSessionResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	BoardID   string `json:"board_id"`
	ExpiresAt string `json:"expires_at"`
}

// @Summary  Start a demo session
// @Tags     auth
// @Produce  json
// @Success  201 {object} demoSessionResponse
// @Failure  429 {object} httputil.ErrorResponse "too many demo sessions from this IP"
// @Failure  500 {object} httputil.ErrorResponse
// @Router   /api/auth/demo [post]
func (h *AuthHandler) Demo(w http.ResponseWriter, r *http.Request) error {
	sandbox, err := h.demoService.CreateSandbox(r.Context(), service.SeedMemberEmail)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to start demo", err)
	}

	if err := h.issueSession(w, r, sandbox.UserID, sandbox.Email); err != nil {
		return err
	}
	httputil.RespondJSON(w, http.StatusCreated, demoSessionResponse{
		ID:        sandbox.UserID,
		Email:     sandbox.Email,
		FullName:  sandbox.FullName,
		BoardID:   sandbox.BoardID,
		ExpiresAt: sandbox.ExpireAt.Format(time.RFC3339),
	})
	return nil
}

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
		// Invalid and expired both return 401, so valid-but-expired tokens can't be probed.
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
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"user_id":         user.ID,
		"email":           user.Email,
		"full_name":       user.FullName,
		"is_demo":         user.IsDemo,
		"demo_expires_at": user.DemoExpiresAt,
	})
	return nil
}

type wsTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
}

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
