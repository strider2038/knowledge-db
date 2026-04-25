package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

// GoogleOAuthStart handles GET /api/auth/google: redirect to Google with signed state.
func (h *AuthHandler) GoogleOAuthStart(w http.ResponseWriter, r *http.Request) {
	if h.cfg.Auth.AuthMode() != config.AuthModeGoogle {
		w.WriteHeader(http.StatusNotFound)

		return
	}
	returnPath := sanitizeReturnPath(r.URL.Query().Get("redirect"))
	st, err := signOAuthState(h.cfg.Auth.OAuthStateSecret, returnPath, time.Now())
	if err != nil {
		clog.Errorf(r.Context(), "auth google: sign state: %v", err)
		writeError(w, http.StatusInternalServerError, "oauth state error")

		return
	}
	loc := h.googleAuthorizationURL(st)
	http.Redirect(w, r, loc, http.StatusFound)
}

// GoogleOAuthCallback handles GET /api/auth/google/callback.
func (h *AuthHandler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.cfg.Auth.AuthMode() != config.AuthModeGoogle {
		w.WriteHeader(http.StatusNotFound)

		return
	}
	q := r.URL.Query()
	if q.Get("error") != "" {
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "oauth")

		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if code == "" {
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "oauth")

		return
	}
	returnPath, err := verifyOAuthState(h.cfg.Auth.OAuthStateSecret, state)
	if err != nil {
		clog.Warn(r.Context(), "auth google: invalid state", "err", err)
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "state")

		return
	}
	email, verified, err := h.exchangeCodeForUserInfo(r.Context(), code)
	if err != nil {
		clog.Warn(r.Context(), "auth google: token or userinfo failed", "err", err)
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "oauth")

		return
	}
	if !verified {
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "forbidden")

		return
	}
	allow := parseEmailAllowlist(h.cfg.Auth.AuthAllowedEmails)
	if !isEmailAllowed(allow, email) {
		clog.Info(r.Context(), "auth google: email not in allowlist", "email", email)
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "forbidden")

		return
	}
	sid, err := h.store.Create(h.cfg.Auth.SessionTTL)
	if err != nil {
		clog.Errorf(r.Context(), "auth google: create session: %w", err)
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "server")

		return
	}
	setSessionCookie(w, r, sid, int(h.cfg.Auth.SessionTTL.Seconds()))
	dest, err := appendQueryPath(h.cfg.WebPublicBaseURL, returnPath, "")
	if err != nil {
		clog.Errorf(r.Context(), "auth google: build redirect: %v", err)
		h.redirectOAuthError(w, r, h.cfg.WebPublicBaseURL, "config")

		return
	}
	clog.Info(r.Context(), "auth google: success", "email", email)
	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *AuthHandler) googleAuthorizationURL(state string) string {
	base := googleAuthURL
	if h.testAuthURL != "" {
		base = h.testAuthURL
	}
	v := url.Values{}
	v.Set("client_id", h.cfg.Auth.GoogleClientID)
	v.Set("redirect_uri", h.cfg.Auth.GoogleRedirectURL)
	v.Set("response_type", "code")
	v.Set("scope", "openid email profile")
	v.Set("state", state)
	v.Set("access_type", "online")

	return base + "?" + v.Encode()
}

//nolint:tagliatelle // Google token endpoint field names
type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

//nolint:tagliatelle // Google userinfo field names
type googleUserInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (h *AuthHandler) tokenEndpoint() string {
	if h.testTokenURL != "" {
		return h.testTokenURL
	}

	return googleTokenURL
}

func (h *AuthHandler) userInfoEndpoint() string {
	if h.testUserInfoURL != "" {
		return h.testUserInfoURL
	}

	return googleUserInfoURL
}

func (h *AuthHandler) exchangeCodeForUserInfo(ctx context.Context, code string) (string, bool, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", h.cfg.Auth.GoogleClientID)
	data.Set("client_secret", h.cfg.Auth.GoogleClientSecret)
	data.Set("redirect_uri", h.cfg.Auth.GoogleRedirectURL)
	data.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.tokenEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return "", false, errors.Errorf("token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := h.httpClient.Do(req)
	if err != nil {
		return "", false, errors.Errorf("token http: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", false, errors.Errorf("token: status %d", res.StatusCode)
	}
	var tr googleTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return "", false, errors.Errorf("token json: %w", err)
	}
	if tr.AccessToken == "" {
		return "", false, errors.New("empty access token from Google")
	}
	uReq, err := http.NewRequestWithContext(ctx, http.MethodGet, h.userInfoEndpoint(), nil)
	if err != nil {
		return "", false, errors.Errorf("userinfo request: %w", err)
	}
	uReq.Header.Set("Authorization", "Bearer "+tr.AccessToken)
	uRes, err := h.httpClient.Do(uReq)
	if err != nil {
		return "", false, errors.Errorf("userinfo http: %w", err)
	}
	defer uRes.Body.Close()
	if uRes.StatusCode != http.StatusOK {
		return "", false, errors.Errorf("userinfo: status %d", uRes.StatusCode)
	}
	var u googleUserInfo
	if err := json.NewDecoder(uRes.Body).Decode(&u); err != nil {
		return "", false, errors.Errorf("userinfo json: %w", err)
	}

	return strings.TrimSpace(u.Email), u.EmailVerified, nil
}

func (h *AuthHandler) redirectOAuthError(w http.ResponseWriter, r *http.Request, publicBase, errCode string) {
	if publicBase == "" {
		http.Redirect(w, r, "/login?error="+url.QueryEscape(errCode), http.StatusFound)

		return
	}
	dest, err := appendQueryPath(publicBase, "/login", "error="+url.QueryEscape(errCode))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "redirect error")

		return
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func sanitizeReturnPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/"
	}
	if strings.HasPrefix(p, "//") {
		return "/"
	}
	if strings.Contains(p, "://") {
		return "/"
	}
	cleaned := path.Clean(p)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		return "/"
	}

	return cleaned
}
