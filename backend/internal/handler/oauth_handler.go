// internal/handler/oauth_handler.go
package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleUserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type OAuthHandler struct {
	googleConfig *oauth2.Config
	authService  service.AuthServicer
	frontendURL  string
	production   bool
	crossSite    bool
}

func NewOAuthHandler(
	clientID, clientSecret, redirectURL, frontendURL string,
	authService service.AuthServicer,
	production, crossSite bool,
) *OAuthHandler {
	return &OAuthHandler{
		googleConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		authService: authService,
		frontendURL: frontendURL,
		production:  production,
		crossSite:   crossSite,
	}
}

// GET /api/auth/google
func (h *OAuthHandler) RedirectToGoogle(w http.ResponseWriter, r *http.Request) error {
	state, err := generateState()
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to generate state", err)
	}
	sameSite, secure := token.AuthCookieSameSite(http.SameSiteLaxMode, h.production, h.crossSite)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
	http.Redirect(w, r, h.googleConfig.AuthCodeURL(state), http.StatusTemporaryRedirect)
	return nil
}

// GET /api/auth/google/callback
func (h *OAuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != r.URL.Query().Get("state") {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid state", nil)
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})

	code := r.URL.Query().Get("code")
	oauthToken, err := h.googleConfig.Exchange(r.Context(), code)
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Failed to exchange token", err)
	}

	userInfo, err := fetchGoogleUserInfo(r.Context(), oauthToken.AccessToken)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to get user info", err)
	}

	user, err := h.authService.UpsertOAuthUser(r.Context(), userInfo.Email, userInfo.Name, "google", userInfo.Sub)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to authenticate", err)
	}

	jwtToken, err := token.Generate(user.ID, user.Email)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to generate token", err)
	}
	token.SetAuthCookie(w, jwtToken, h.production, h.crossSite)

	refreshRaw, err := h.authService.IssueRefreshToken(r.Context(), user.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		// Non-fatal: user still gets a 15-minute session via the access cookie
		// and will be bounced to login when it expires.
		slog.Error("issue refresh token on oauth", "user_id", user.ID, "err", err)
	} else {
		token.SetRefreshCookie(w, refreshRaw, h.production, h.crossSite)
	}

	http.Redirect(w, r, h.frontendURL+"/auth/callback", http.StatusTemporaryRedirect)
	return nil
}

// fetchGoogleUserInfo retrieves the authenticated user's profile using the Bearer token.
func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api returned status %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}
	return &info, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
