package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/auth"
	"github.com/strider2038/knowledge-db/internal/auth/session"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/mcp"
)

const testLoginBody = `{"login":"testuser","password":"testpass"}`

func setupAuthTestHandler(t *testing.T, authEnabled bool) http.Handler {
	t.Helper()
	dataPath := t.TempDir()
	uploadsDir := t.TempDir()
	h := api.NewHandlerWithUploads(dataPath, uploadsDir, &ingestion.StubIngester{}, nil)

	cfg := &config.Config{
		DataPath:   dataPath,
		UploadsDir: uploadsDir,
		HTTP:       config.HTTP{},
		Auth: config.Auth{
			Login:      "",
			Password:   "",
			SessionTTL: 8 * time.Hour,
		},
	}
	if authEnabled {
		cfg.Auth.Login = "testuser"
		cfg.Auth.Password = "testpass"
	}

	store := session.NewStore()
	authHandler := api.NewAuthHandler(store, cfg)
	mux, err := api.NewMux(h, authHandler)
	require.NoError(t, err)
	mux.Handle("GET /api/mcp", mcp.NewHandler(dataPath))
	mux.Handle("POST /api/mcp", mcp.NewHandler(dataPath))

	handler := api.Gzip(api.CORS(mux, ""))
	if authEnabled {
		handler = auth.Middleware(handler, store)
	}

	return handler
}

func TestAuthLogin_WhenValidCredentials_ExpectSessionCookie(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	resp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))

	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("authenticated").IsTrue()
	})
	assert.NotEmpty(t, resp.Header().Get("Set-Cookie"))
}

func TestAuthLogin_WhenInvalidCredentials_Expect401(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	body := `{"login":"wrong","password":"wrong"}`
	resp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(body)),
		apitest.WithContentType("application/json"))

	resp.IsUnauthorized()
}

func TestAuthLogin_WhenAuthDisabled_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, false)

	resp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))

	resp.IsBadRequest()
}

func TestAuthSession_WhenNoCookie_ExpectAuthenticatedFalse(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	resp := apitest.HandleGET(t, handler, "/api/auth/session")
	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("authenticated").IsFalse()
	})
}

func TestAuthSession_WhenValidCookie_ExpectAuthenticatedTrue(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	loginResp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))
	loginResp.IsOK()

	cookie := loginResp.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var sess struct {
		Authenticated bool `json:"authenticated"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&sess))
	assert.True(t, sess.Authenticated)
}

func TestAuthSession_WhenAuthDisabled_ExpectAuthenticatedTrue(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, false)

	resp := apitest.HandleGET(t, handler, "/api/auth/session")
	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("authenticated").IsTrue()
	})
}

func TestAuthLogout_WhenValidSession_ExpectCookieCleared(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	loginResp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))
	cookie := loginResp.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	setCookie := rec.Header().Get("Set-Cookie")
	assert.True(t, strings.Contains(setCookie, "Max-Age=-1") || strings.Contains(setCookie, "Max-Age=0"),
		"cookie should be cleared: %s", setCookie)
}

func TestProtectedAPI_WhenAuthEnabledAndNoSession_Expect401(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	resp := apitest.HandleGET(t, handler, "/api/tree")
	resp.IsUnauthorized()
}

func TestProtectedAPI_WhenAuthEnabledAndValidSession_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	loginResp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))
	cookie := loginResp.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	req := httptest.NewRequest(http.MethodGet, "/api/tree", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestProtectedAPI_WhenAuthDisabled_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, false)

	resp := apitest.HandleGET(t, handler, "/api/tree")
	resp.IsOK()
}

func TestAuthAllowlist_WhenAuthEnabled_HealthzWithoutSession_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	resp := apitest.HandleGET(t, handler, "/healthz")
	resp.IsOK()
}

func TestMCP_WhenAuthEnabledAndNoSession_Expect401(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	resp := apitest.HandleGET(t, handler, "/api/mcp")
	resp.IsUnauthorized()
}

func TestMCP_WhenAuthEnabledAndValidSession_ExpectReachesHandler(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	loginResp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))
	cookie := loginResp.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	req := httptest.NewRequest(http.MethodGet, "/api/mcp", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// MCP handler returns 501, but we get past auth (not 401)
	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestAssets_WhenAuthEnabledAndNoSession_Expect401(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	resp := apitest.HandleGET(t, handler, "/api/assets/topic/node1/image.png")
	resp.IsUnauthorized()
}

func TestAssets_WhenAuthEnabledAndValidSession_ExpectOKOr404(t *testing.T) {
	t.Parallel()
	handler := setupAuthTestHandler(t, true)

	loginResp := apitest.HandlePOST(t, handler, "/api/auth/login", bytes.NewReader([]byte(testLoginBody)),
		apitest.WithContentType("application/json"))
	cookie := loginResp.Header().Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	req := httptest.NewRequest(http.MethodGet, "/api/assets/topic/node1/nonexistent.png", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should get past auth (not 401); may get 404 for missing asset
	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
}
