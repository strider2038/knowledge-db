package api

import (
	"net/http"
	"time"

	"github.com/muonsoft/clog"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/googleoauth"
)

func googleOAuthConfigFromApp(cfg *config.Config) googleoauth.Config {
	return googleoauth.Config{
		ClientID:      cfg.Auth.GoogleClientID,
		ClientSecret:  cfg.Auth.GoogleClientSecret,
		RedirectURL:   cfg.Auth.GoogleRedirectURL,
		StateSecret:   cfg.Auth.OAuthStateSecret,
		AllowedEmails: cfg.Auth.AuthAllowedEmails,
	}
}

// GoogleOAuthStart handles GET /api/auth/google: redirect to Google with signed state.
func (h *AuthHandler) GoogleOAuthStart(w http.ResponseWriter, r *http.Request) {
	if h.cfg.Auth.AuthMode() != config.AuthModeGoogle {
		w.WriteHeader(http.StatusNotFound)

		return
	}
	returnPath := googleoauth.SanitizeReturnPath(r.URL.Query().Get("redirect"))
	st, err := googleoauth.SignState(h.googleClient.Config.StateSecret, returnPath, time.Now())
	if err != nil {
		clog.Errorf(r.Context(), "auth google: sign state: %v", err)
		writeError(w, http.StatusInternalServerError, "oauth state error")

		return
	}
	loc := h.googleClient.AuthorizationURL(st)
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
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "oauth")

		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if code == "" {
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "oauth")

		return
	}
	returnPath, err := googleoauth.VerifyState(h.googleClient.Config.StateSecret, state)
	if err != nil {
		clog.Warn(r.Context(), "auth google: invalid state", "err", err)
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "state")

		return
	}
	email, verified, err := h.googleClient.ExchangeCodeForUserInfo(r.Context(), code)
	if err != nil {
		clog.Warn(r.Context(), "auth google: token or userinfo failed", "err", err)
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "oauth")

		return
	}
	if !verified {
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "forbidden")

		return
	}
	allow := googleoauth.ParseEmailAllowlist(h.googleClient.Config.AllowedEmails)
	if !googleoauth.IsEmailAllowed(allow, email) {
		clog.Info(r.Context(), "auth google: email not in allowlist", "email", email)
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "forbidden")

		return
	}
	sid, err := h.store.Create(h.cfg.Auth.SessionTTL)
	if err != nil {
		clog.Errorf(r.Context(), "auth google: create session: %w", err)
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "server")

		return
	}
	setSessionCookie(w, r, sid, int(h.cfg.Auth.SessionTTL.Seconds()))
	dest, err := googleoauth.AppendQueryPath(h.cfg.WebPublicBaseURL, returnPath, "")
	if err != nil {
		clog.Errorf(r.Context(), "auth google: build redirect: %v", err)
		googleoauth.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "config")

		return
	}
	clog.Info(r.Context(), "auth google: success", "email", email)
	http.Redirect(w, r, dest, http.StatusFound)
}
