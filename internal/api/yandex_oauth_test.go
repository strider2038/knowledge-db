package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/auth"
	"github.com/strider2038/knowledge-db/internal/auth/session"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/oauthcommon"
)

func newYandexE2EHandler(t *testing.T, userInfoJSON, allowlist, webBase string) http.Handler {
	t.Helper()
	muxO := http.NewServeMux()
	muxO.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ok","token_type":"bearer","expires_in":3600}`))
	})
	muxO.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(userInfoJSON))
	})
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
			YandexClientID:     "c",
			YandexClientSecret: "s",
			YandexRedirectURL:  redirect,
			AuthAllowedEmails:  allowlist,
			OAuthStateSecret:   testOAuthStateSecret,
			SessionTTL:         8 * time.Hour,
		},
	}
	require.NoError(t, cfg.Auth.ValidateAuth())
	require.NoError(t, cfg.Auth.ValidateWebPublicBaseForOAuth(cfg.WebPublicBaseURL))
	st := session.NewStore()
	ah := NewAuthHandler(st, cfg)
	ah.setTestYandexOAuthServers(mock.Client(), mock.URL+"/authz", mock.URL+"/token", mock.URL+"/userinfo")
	mh := NewHandlerWithUploads(data, up, &ingestion.StubIngester{}, nil)
	router, err := NewMux(mh, ah)
	require.NoError(t, err)

	return auth.Middleware(Gzip(CORS(router, "")), st)
}

func TestYandexOAuthStart_WhenNotConfigured_Expect404(t *testing.T) {
	t.Parallel()
	h := newPasswordOnlyE2EHandler(t)
	r := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/yandex", nil)
	require.NoError(t, err)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusNotFound, r.Code)
}

func TestYandexCallback_EmptyEmail_ExpectRedirectWithProvider(t *testing.T) {
	t.Parallel()
	h := newYandexE2EHandler(t, `{"default_email":""}`, "a@ex.com", "http://web:9")
	signed, err := oauthcommon.SignState(testOAuthStateSecret, "/", time.Now())
	require.NoError(t, err)
	u := "/api/auth/yandex/callback?code=ok&state=" + url.QueryEscape(signed)
	r := httptest.NewRecorder()
	req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	require.NoError(t, rerr)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusFound, r.Code)
	loc, perr := url.Parse(r.Header().Get("Location"))
	require.NoError(t, perr)
	assert.Equal(t, "forbidden", loc.Query().Get("error"))
	assert.Equal(t, "yandex", loc.Query().Get("provider"))
}

func TestYandexCallback_ForbiddenEmail_ExpectRedirectWithProvider(t *testing.T) {
	t.Parallel()
	h := newYandexE2EHandler(t, `{"default_email":"b@ex.com"}`, "only@a.com", "http://web:9")
	signed, err := oauthcommon.SignState(testOAuthStateSecret, "/", time.Now())
	require.NoError(t, err)
	u := "/api/auth/yandex/callback?code=ok&state=" + url.QueryEscape(signed)
	r := httptest.NewRecorder()
	req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	require.NoError(t, rerr)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusFound, r.Code)
	loc, perr := url.Parse(r.Header().Get("Location"))
	require.NoError(t, perr)
	assert.Equal(t, "forbidden", loc.Query().Get("error"))
	assert.Equal(t, "yandex", loc.Query().Get("provider"))
}

func TestYandexCallback_OK_ExpectSessionCookie(t *testing.T) {
	t.Parallel()
	web := "http://127.0.0.1:8"
	h := newYandexE2EHandler(t, `{"default_email":"a@ex.com"}`, "a@ex.com", web)
	signed, err := oauthcommon.SignState(testOAuthStateSecret, "/tree", time.Now())
	require.NoError(t, err)
	u := "/api/auth/yandex/callback?code=ok&state=" + url.QueryEscape(signed)
	rec := httptest.NewRecorder()
	req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	require.NoError(t, rerr)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	loc, perr := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, perr)
	assert.Equal(t, "/tree", loc.Path)
	assert.Contains(t, rec.Header().Get("Set-Cookie"), session.CookieName)
}

func TestGetSession_YandexMode_IncludesAuthMethods(t *testing.T) {
	t.Parallel()
	h := newYandexE2EHandler(t, `{"default_email":"a@ex.com"}`, "a@ex.com", "http://w/")
	r := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/session", nil)
	require.NoError(t, err)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusOK, r.Code)
	var out map[string]any
	require.NoError(t, json.NewDecoder(r.Body).Decode(&out))
	assert.Equal(t, "yandex", out["auth_mode"])
	methods, ok := out["auth_methods"].([]any)
	require.True(t, ok)
	require.Len(t, methods, 1)
	assert.Equal(t, "yandex", methods[0])
}

func TestGetSession_GoogleAndYandex_IncludesAuthMethods(t *testing.T) {
	t.Parallel()
	data := t.TempDir()
	up := t.TempDir()
	cfg := &config.Config{
		DataPath:         data,
		UploadsDir:       up,
		WebPublicBaseURL: "http://web/",
		Auth: config.Auth{
			GoogleClientID:     "g",
			GoogleClientSecret: "gs",
			GoogleRedirectURL:  "http://localhost/gcb",
			YandexClientID:     "y",
			YandexClientSecret: "ys",
			YandexRedirectURL:  "http://localhost/ycb",
			AuthAllowedEmails:  "a@ex.com",
			OAuthStateSecret:   testOAuthStateSecret,
			SessionTTL:         8 * time.Hour,
		},
	}
	require.NoError(t, cfg.Auth.ValidateAuth())
	require.NoError(t, cfg.Auth.ValidateWebPublicBaseForOAuth(cfg.WebPublicBaseURL))
	st := session.NewStore()
	ah := NewAuthHandler(st, cfg)
	mh := NewHandlerWithUploads(data, up, &ingestion.StubIngester{}, nil)
	router, err := NewMux(mh, ah)
	require.NoError(t, err)
	h := auth.Middleware(Gzip(CORS(router, "")), st)

	r := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/session", nil)
	require.NoError(t, err)
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusOK, r.Code)
	var out map[string]any
	require.NoError(t, json.NewDecoder(r.Body).Decode(&out))
	assert.Equal(t, "multi", out["auth_mode"])
	methods, ok := out["auth_methods"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"google", "yandex"}, methods)
}
