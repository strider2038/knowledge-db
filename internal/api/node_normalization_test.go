package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/muonsoft/errors"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type mockNodeNormalizer struct {
	err error
}

func (m *mockNodeNormalizer) NormalizeNode(_ context.Context, _ string, _ *kb.Node, onLog func(stream, text string)) error {
	onLog("stdout", "line one")
	onLog("stderr", "line two")
	return m.err
}

type normalizeResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Stage     string `json:"stage"`
	SyncDone  bool   `json:"sync_done"`
	ErrorText string `json:"error"`
}

func setupNormalizeHandler(t *testing.T, normalizer *mockNodeNormalizer, committer *mockGitCommitter, gitDisabled bool) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := "---\nkeywords: [test]\n---\n\nContent"
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetNodeNormalizer(normalizer)
	h.SetGitCommitter(committer, nil, gitDisabled)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func doJSON(t *testing.T, mux http.Handler, method, path string) (int, normalizeResponse) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out normalizeResponse
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec.Code, out
}

func TestPostNodeNormalize_WhenSuccess_ExpectOperationSuccess(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{}, false)

	code, start := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/normalize")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)
	require.Equal(t, "running", start.Status)

	require.Eventually(t, func() bool {
		stCode, st := doJSON(t, mux, http.MethodGet, "/api/node-normalization/"+start.ID)
		return stCode == http.StatusOK && st.Status == "success" && st.SyncDone
	}, time.Second, 20*time.Millisecond)

	logsReq := httptest.NewRequest(http.MethodGet, "/api/node-normalization/"+start.ID+"/logs", nil)
	logsRec := httptest.NewRecorder()
	mux.ServeHTTP(logsRec, logsReq)
	require.Equal(t, http.StatusOK, logsRec.Code)
	require.Contains(t, logsRec.Body.String(), "line one")
}

func TestPostNodeNormalize_WhenNormalizerFails_ExpectOperationError(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{err: errors.New("normalize failed")}, &mockGitCommitter{}, false)

	code, start := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/normalize")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		stCode, st := doJSON(t, mux, http.MethodGet, "/api/node-normalization/"+start.ID)
		return stCode == http.StatusOK && st.Status == "error" && st.Stage == "normalize"
	}, time.Second, 20*time.Millisecond)
}

func TestPostNodeNormalize_WhenSyncFails_ExpectOperationErrorOnSyncStage(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{syncErr: errors.New("sync failed")}, false)

	code, start := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/normalize")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		stCode, st := doJSON(t, mux, http.MethodGet, "/api/node-normalization/"+start.ID)
		return stCode == http.StatusOK && st.Status == "error" && st.Stage == "sync"
	}, time.Second, 20*time.Millisecond)
}

func TestPostNodeNormalize_WhenNoRunner_Expect503(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	code, _ := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/normalize")
	require.Equal(t, http.StatusServiceUnavailable, code)
}
