package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/muonsoft/clog"
	"github.com/strider2038/knowledge-db/internal/auth/session"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/googleoauth"
)

const (
	rateLimit  = 3
	rateWindow = 60 * time.Minute
)

// AuthHandler — handlers для auth endpoints.
type AuthHandler struct {
	store             *session.Store
	cfg               *config.Config
	allowedCORSOrigin string
	rateMu            sync.Mutex
	rateMap           map[string][]time.Time
	googleClient      *googleoauth.Client
}

// NewAuthHandler создаёт AuthHandler.
func NewAuthHandler(store *session.Store, cfg *config.Config) *AuthHandler {
	gc := &googleoauth.Client{
		Config: googleOAuthConfigFromApp(cfg),
		HTTPClient: &http.Client{
			Timeout: googleoauth.DefaultOutboundTimeout,
		},
	}
	h := &AuthHandler{
		store:             store,
		cfg:               cfg,
		allowedCORSOrigin: cfg.HTTP.AllowedCORSOrigin,
		rateMap:           make(map[string][]time.Time),
		googleClient:      gc,
	}
	go h.cleanupRateMap()

	return h
}

// Login обрабатывает POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	if h.cfg.Auth.AuthMode() == config.AuthModeOff {
		writeError(w, http.StatusBadRequest, "auth disabled")

		return
	}
	if h.cfg.Auth.AuthMode() == config.AuthModeGoogle {
		writeError(w, http.StatusBadRequest, "use Google sign-in")

		return
	}

	if !h.validateOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")

		return
	}

	ip := clientIP(r)
	if h.isRateLimited(ip) {
		clog.Warn(r.Context(), "auth login: rate limited", "ip", ip)
		w.WriteHeader(http.StatusTooManyRequests)

		return
	}

	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")

		return
	}

	loginOK := subtle.ConstantTimeCompare([]byte(h.cfg.Auth.Login), []byte(req.Login)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(h.cfg.Auth.Password), []byte(req.Password)) == 1

	if !loginOK || !passOK {
		h.recordFailedAttempt(ip)
		clog.Warn(r.Context(), "auth login: invalid credentials")
		writeError(w, http.StatusUnauthorized, "invalid credentials")

		return
	}

	sessionID, err := h.store.Create(h.cfg.Auth.SessionTTL)
	if err != nil {
		clog.Errorf(r.Context(), "auth login: session creation failed: %w", err)
		writeError(w, http.StatusInternalServerError, "session creation failed")

		return
	}

	clog.Info(r.Context(), "auth login: success")
	setSessionCookie(w, r, sessionID, int(h.cfg.Auth.SessionTTL.Seconds()))
	writeJSON(w, map[string]bool{"authenticated": true})
}

// Session обрабатывает GET /api/auth/session.
func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	if h.cfg.Auth.AuthMode() == config.AuthModeOff {
		writeJSON(w, map[string]any{
			"authenticated": true,
			"auth_enabled":  false,
		})

		return
	}

	base := map[string]any{
		"auth_enabled": true,
		"auth_mode":    string(h.cfg.Auth.AuthMode()),
	}
	cookie, err := r.Cookie(session.CookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		base["authenticated"] = false
		writeJSON(w, base)

		return
	}

	authenticated := h.store.Get(cookie.Value)
	base["authenticated"] = authenticated
	writeJSON(w, base)
}

// Logout обрабатывает POST /api/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	if !h.validateOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid origin")

		return
	}

	if h.cfg.Auth.AuthEnabled() {
		cookie, _ := r.Cookie(session.CookieName)
		if cookie != nil && cookie.Value != "" {
			h.store.Invalidate(cookie.Value)
		}
	}

	clog.Info(r.Context(), "auth logout: success")
	clearSessionCookie(w, r)
	writeJSON(w, map[string]bool{"authenticated": false})
}

func (h *AuthHandler) setTestGoogleOAuthServers(c *http.Client, authURL, tokenURL, userInfoURL string) {
	h.googleClient.HTTPClient = c
	h.googleClient.Endpoints = googleoauth.Endpoints{
		AuthURL:     authURL,
		TokenURL:    tokenURL,
		UserInfoURL: userInfoURL,
	}
}

func (h *AuthHandler) isRateLimited(ip string) bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateWindow)
	attempts := h.rateMap[ip]

	// Prune old attempts
	var valid []time.Time
	for _, t := range attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	h.rateMap[ip] = valid

	return len(valid) >= rateLimit
}

func (h *AuthHandler) recordFailedAttempt(ip string) {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()

	h.rateMap[ip] = append(h.rateMap[ip], time.Now())
}

func (h *AuthHandler) cleanupRateMap() {
	ticker := time.NewTicker(rateWindow)
	defer ticker.Stop()

	for range ticker.C {
		h.pruneAllRateEntries()
	}
}

func (h *AuthHandler) pruneAllRateEntries() {
	cutoff := time.Now().Add(-rateWindow)

	h.rateMu.Lock()
	defer h.rateMu.Unlock()

	for ip, attempts := range h.rateMap {
		var valid []time.Time
		for _, t := range attempts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(h.rateMap, ip)
		} else {
			h.rateMap[ip] = valid
		}
	}
}

func (h *AuthHandler) validateOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if h.allowedCORSOrigin == "" {
		return true
	}

	return origin == h.allowedCORSOrigin
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return strings.TrimSpace(strings.Split(x, ",")[0])
	}

	return r.RemoteAddr
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string, maxAge int) {
	cookie := &http.Cookie{
		Name:     session.CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if isSecureRequest(r) {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if isSecureRequest(r) {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)
}
