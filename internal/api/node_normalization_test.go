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
	ID       string `json:"id"`
	Status   string `json:"status"`
	Stage    string `json:"stage"`
	SyncDone bool   `json:"sync_done"`
	Error    string `json:"error"`
}

type normalizeLogsResponse struct {
	Entries []struct {
		Offset int64  `json:"offset"`
		Stream string `json:"stream"`
		Text   string `json:"text"`
	} `json:"entries"`
	NextOffset int64 `json:"next_offset"`
}

func setupNormalizeHandler(t *testing.T, normalizer *mockNodeNormalizer, committer *mockGitCommitter) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := "---\nkeywords: [test]\n---\n\nContent"
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetNodeNormalizer(normalizer)
	h.SetGitCommitter(committer, nil, false)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func doJSON(t *testing.T, mux http.Handler, method, path string) (int, normalizeResponse) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
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
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{})

	code, start := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/normalize")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)
	require.Equal(t, "running", start.Status)

	require.Eventually(t, func() bool {
		stCode, st := doJSON(t, mux, http.MethodGet, "/api/node-normalization/"+start.ID)

		return stCode == http.StatusOK && st.Status == "success" && st.SyncDone
	}, time.Second, 20*time.Millisecond)

	logsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-normalization/"+start.ID+"/logs", nil)
	logsRec := httptest.NewRecorder()
	mux.ServeHTTP(logsRec, logsReq)
	require.Equal(t, http.StatusOK, logsRec.Code)
	require.Contains(t, logsRec.Body.String(), "line one")
}

func TestPostNodeNormalize_WhenNodeMissing_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{})

	code, _ := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/missing/normalize")
	require.Equal(t, http.StatusNotFound, code)
}

func TestGetNodeNormalizeLogs_WhenAfterProvided_ExpectOnlyNewEntries(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{})

	code, start := doJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/normalize")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		stCode, st := doJSON(t, mux, http.MethodGet, "/api/node-normalization/"+start.ID)

		return stCode == http.StatusOK && st.Status == "success"
	}, time.Second, 20*time.Millisecond)

	reqAll := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-normalization/"+start.ID+"/logs", nil)
	recAll := httptest.NewRecorder()
	mux.ServeHTTP(recAll, reqAll)
	require.Equal(t, http.StatusOK, recAll.Code)

	var all normalizeLogsResponse
	require.NoError(t, json.Unmarshal(recAll.Body.Bytes(), &all))
	require.GreaterOrEqual(t, len(all.Entries), 4)
	require.GreaterOrEqual(t, all.NextOffset, int64(len(all.Entries)))

	reqAfter := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-normalization/"+start.ID+"/logs?after=2", nil)
	recAfter := httptest.NewRecorder()
	mux.ServeHTTP(recAfter, reqAfter)
	require.Equal(t, http.StatusOK, recAfter.Code)

	var filtered normalizeLogsResponse
	require.NoError(t, json.Unmarshal(recAfter.Body.Bytes(), &filtered))
	for _, entry := range filtered.Entries {
		require.Greater(t, entry.Offset, int64(2))
	}
}

func TestGetNodeNormalizeLogs_WhenUnknownOperation_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-normalization/missing/logs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostNodeNormalize_WhenNormalizerFails_ExpectOperationError(t *testing.T) {
	t.Parallel()
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{err: errors.New("normalize failed")}, &mockGitCommitter{})

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
	mux := setupNormalizeHandler(t, &mockNodeNormalizer{}, &mockGitCommitter{syncErr: errors.New("sync failed")})

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

func TestPostNodeNormalize_WhenCursorAgentMissing_Expect503(t *testing.T) {
	t.Setenv("PATH", "")
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := "---\nkeywords: [test]\n---\n\nContent"
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetNodeNormalizer(api.NewCursorNodeNormalizer())
	h.SetGitCommitter(&mockGitCommitter{}, nil, false)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/nodes/topic/my-node/normalize", nil)
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "cursor-agent not found in PATH")
}
