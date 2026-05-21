package api

import (
	"net/http"
	"time"

	"github.com/muonsoft/clog"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/oauthcommon"
	"github.com/strider2038/knowledge-db/internal/yandexoauth"
)

func yandexOAuthConfigFromApp(cfg *config.Config) yandexoauth.Config {
	return yandexoauth.Config{
		ClientID:      cfg.Auth.YandexClientID,
		ClientSecret:  cfg.Auth.YandexClientSecret,
		RedirectURL:   cfg.Auth.YandexRedirectURL,
		StateSecret:   cfg.Auth.OAuthStateSecret,
		AllowedEmails: cfg.Auth.AuthAllowedEmails,
	}
}

// YandexOAuthStart handles GET /api/auth/yandex: redirect to Yandex with signed state.
func (h *AuthHandler) YandexOAuthStart(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Auth.YandexAuthConfigured() {
		w.WriteHeader(http.StatusNotFound)

		return
	}
	returnPath := oauthcommon.SanitizeReturnPath(r.URL.Query().Get("redirect"))
	st, err := oauthcommon.SignState(h.yandexClient.Config.StateSecret, returnPath, time.Now())
	if err != nil {
		clog.Errorf(r.Context(), "auth yandex: sign state: %v", err)
		writeError(w, http.StatusInternalServerError, "oauth state error")

		return
	}
	loc := h.yandexClient.AuthorizationURL(st)
	http.Redirect(w, r, loc, http.StatusFound)
}

// YandexOAuthCallback handles GET /api/auth/yandex/callback.
func (h *AuthHandler) YandexOAuthCallback(w http.ResponseWriter, r *http.Request) {
	const provider = "yandex"
	if !h.cfg.Auth.YandexAuthConfigured() {
		w.WriteHeader(http.StatusNotFound)

		return
	}
	q := r.URL.Query()
	if q.Get("error") != "" {
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "oauth", provider)

		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if code == "" {
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "oauth", provider)

		return
	}
	returnPath, err := oauthcommon.VerifyState(h.yandexClient.Config.StateSecret, state)
	if err != nil {
		clog.Warn(r.Context(), "auth yandex: invalid state", "err", err)
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "state", provider)

		return
	}
	email, err := h.yandexClient.ExchangeCodeForUserInfo(r.Context(), code)
	if err != nil {
		clog.Warn(r.Context(), "auth yandex: token or userinfo failed", "err", err)
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "oauth", provider)

		return
	}
	if email == "" {
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "forbidden", provider)

		return
	}
	allow := oauthcommon.ParseEmailAllowlist(h.yandexClient.Config.AllowedEmails)
	if !oauthcommon.IsEmailAllowed(allow, email) {
		clog.Info(r.Context(), "auth yandex: email not in allowlist", "email", email)
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "forbidden", provider)

		return
	}
	sid, err := h.store.Create(h.cfg.Auth.SessionTTL)
	if err != nil {
		clog.Errorf(r.Context(), "auth yandex: create session: %w", err)
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "server", provider)

		return
	}
	setSessionCookie(w, r, sid, int(h.cfg.Auth.SessionTTL.Seconds()))
	dest, err := oauthcommon.AppendQueryPath(h.cfg.WebPublicBaseURL, returnPath, "")
	if err != nil {
		clog.Errorf(r.Context(), "auth yandex: build redirect: %v", err)
		oauthcommon.RedirectToLoginError(w, r, h.cfg.WebPublicBaseURL, "config", provider)

		return
	}
	clog.Info(r.Context(), "auth yandex: success", "email", email)
	http.Redirect(w, r, dest, http.StatusFound)
}
