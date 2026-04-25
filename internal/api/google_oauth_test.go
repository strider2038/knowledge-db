//nolint:testpackage // need access to AuthHandler test hooks; external tests use api package only
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/auth"
	"github.com/strider2038/knowledge-db/internal/auth/session"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/googleoauth"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/mcp"
)

const testOAuthStateSecret = "0123456789abcdef0123456789abcdef" //nolint:gosec // test

func newGoogleE2EHandler(t *testing.T, userInfoJSON, allowlist, webBase string) http.Handler {
	t.Helper()
	muxO := http.NewServeMux()
	muxO.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ok","token_type":"Bearer","expires_in":3600}`))
	})
	muxO.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(userInfoJSON))
	})
	muxO.HandleFunc("/authz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mock := httptest.NewServer(muxO)
	t.Cleanup(mock.Close)

	if webBase == "" {
		webBase = "http://web:9"
	}
	data := t.TempDir()
	up := t.TempDir()
	redirect := mock.URL + "/c"
	cfg := &config.Config{
		DataPath:         data,
		UploadsDir:       up,
		WebPublicBaseURL: webBase,
		Auth: config.Auth{
			GoogleClientID:     "c",
			GoogleClientSecret: "s",
			GoogleRedirectURL:  redirect,
			AuthAllowedEmails:  allowlist,
			OAuthStateSecret:   testOAuthStateSecret,
			SessionTTL:         8 * time.Hour,
		},
	}
	require.NoError(t, cfg.Auth.ValidateAuth())
	require.NoError(t, cfg.Auth.ValidateWebPublicBaseForGoogle(cfg.WebPublicBaseURL))
	st := session.NewStore()
	ah := NewAuthHandler(st, cfg)
	ah.setTestGoogleOAuthServers(mock.Client(), mock.URL+"/authz", mock.URL+"/token", mock.URL+"/userinfo")
	mh := NewHandlerWithUploads(data, up, &ingestion.StubIngester{}, nil)
	router, err := NewMux(mh, ah)
	require.NoError(t, err)
	router.Handle("GET /api/mcp", mcp.NewHandler(data))
	router.Handle("POST /api/mcp", mcp.NewHandler(data))

	return auth.Middleware(Gzip(CORS(router, "")), st)
}

func TestGoogleCallback_InvalidState_ExpectRedirect(t *testing.T) {
	t.Parallel()
	h := newGoogleE2EHandler(t, `{"email":"a@ex.com","email_verified":true}`, "a@ex.com", "http://web:9")
	r := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/google/callback?code=x&state=bad", nil)
	require.NoError(t, err)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusFound, r.Code)
	loc, perr := url.Parse(r.Header().Get("Location"))
	require.NoError(t, perr)
	assert.Equal(t, "web:9", loc.Host)
	assert.Equal(t, "/login", loc.Path)
	assert.Equal(t, "state", loc.Query().Get("error"))
}

func TestGoogleCallback_ForbiddenEmail_ExpectRedirect(t *testing.T) {
	t.Parallel()
	h := newGoogleE2EHandler(t, `{"email":"b@ex.com","email_verified":true}`, "only@a.com", "http://web:9")
	signed, err := googleoauth.SignState(testOAuthStateSecret, "/", time.Now())
	require.NoError(t, err)
	u := "/api/auth/google/callback?code=ok&state=" + url.QueryEscape(signed)
	r := httptest.NewRecorder()
	req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	require.NoError(t, rerr)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusFound, r.Code)
	loc, perr := url.Parse(r.Header().Get("Location"))
	require.NoError(t, perr)
	assert.Equal(t, "forbidden", loc.Query().Get("error"))
}

func TestGoogleCallback_OK_ExpectSessionCookie(t *testing.T) {
	t.Parallel()
	web := "http://127.0.0.1:7"
	h := newGoogleE2EHandler(t, `{"email":"a@ex.com","email_verified":true}`, "a@ex.com", web)
	signed, err := googleoauth.SignState(testOAuthStateSecret, "/add", time.Now())
	require.NoError(t, err)
	u := "/api/auth/google/callback?code=ok&state=" + url.QueryEscape(signed)
	rec := httptest.NewRecorder()
	req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	require.NoError(t, rerr)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	loc, perr := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, perr)
	assert.Equal(t, "127.0.0.1:7", loc.Host)
	assert.Equal(t, "/add", loc.Path)
	assert.Contains(t, rec.Header().Get("Set-Cookie"), session.CookieName)
}

func TestGetSession_GoogleMode_IncludesAuthMode(t *testing.T) {
	t.Parallel()
	h := newGoogleE2EHandler(t, `{"email":"a@ex.com","email_verified":true}`, "a@ex.com", "http://w/")
	r := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/session", nil)
	require.NoError(t, err)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusOK, r.Code)
	var out map[string]any
	require.NoError(t, json.NewDecoder(r.Body).Decode(&out))
	assert.Equal(t, true, out["auth_enabled"])
	assert.Equal(t, "google", out["auth_mode"])
}

func TestPostLogin_GoogleMode_Expect400(t *testing.T) {
	t.Parallel()
	h := newGoogleE2EHandler(t, `{"email":"a@ex.com","email_verified":true}`, "a@ex.com", "http://w/")
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/auth/login", strings.NewReader(`{"login":"a","password":"b"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	r := httptest.NewRecorder()
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusBadRequest, r.Code)
}

