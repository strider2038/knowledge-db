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
)

type dumpImagesResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Stage    string `json:"stage"`
	SyncDone bool   `json:"sync_done"`
	Error    string `json:"error"`
}

type dumpImagesLogsResponse struct {
	Entries []struct {
		Offset int64  `json:"offset"`
		Stream string `json:"stream"`
		Text   string `json:"text"`
	} `json:"entries"`
	NextOffset int64 `json:"next_offset"`
}

func setupDumpImagesHandler(t *testing.T, nodeType string, committer *mockGitCommitter) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := "---\ntype: " + nodeType + "\nkeywords: [test]\n---\n\nContent"
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetGitCommitter(committer, nil, false)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func doDumpJSON(t *testing.T, mux http.Handler, method, path string) (int, dumpImagesResponse) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out dumpImagesResponse
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}

	return rec.Code, out
}

func TestPostNodeDumpImages_WhenSuccess_ExpectOperationSuccess(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "article", &mockGitCommitter{})

	code, start := doDumpJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/dump-images")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)
	require.Equal(t, "running", start.Status)

	require.Eventually(t, func() bool {
		stCode, st := doDumpJSON(t, mux, http.MethodGet, "/api/node-dump-images/"+start.ID)

		return stCode == http.StatusOK && st.Status == "success" && st.SyncDone
	}, time.Second, 20*time.Millisecond)

	logsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-dump-images/"+start.ID+"/logs", nil)
	logsRec := httptest.NewRecorder()
	mux.ServeHTTP(logsRec, logsReq)
	require.Equal(t, http.StatusOK, logsRec.Code)
	require.Contains(t, logsRec.Body.String(), "dump images started")
}

func TestPostNodeDumpImages_WhenNodeMissing_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "article", &mockGitCommitter{})

	code, _ := doDumpJSON(t, mux, http.MethodPost, "/api/nodes/topic/missing/dump-images")
	require.Equal(t, http.StatusNotFound, code)
}

func TestPostNodeDumpImages_WhenNodeNotArticle_Expect400(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "note", &mockGitCommitter{})

	code, _ := doDumpJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/dump-images")
	require.Equal(t, http.StatusBadRequest, code)
}

func TestGetNodeDumpImagesLogs_WhenAfterProvided_ExpectOnlyNewEntries(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "article", &mockGitCommitter{})

	code, start := doDumpJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/dump-images")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		stCode, st := doDumpJSON(t, mux, http.MethodGet, "/api/node-dump-images/"+start.ID)

		return stCode == http.StatusOK && st.Status == "success"
	}, time.Second, 20*time.Millisecond)

	reqAfter := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-dump-images/"+start.ID+"/logs?after=1", nil)
	recAfter := httptest.NewRecorder()
	mux.ServeHTTP(recAfter, reqAfter)
	require.Equal(t, http.StatusOK, recAfter.Code)

	var filtered dumpImagesLogsResponse
	require.NoError(t, json.Unmarshal(recAfter.Body.Bytes(), &filtered))
	for _, entry := range filtered.Entries {
		require.Greater(t, entry.Offset, int64(1))
	}
}

func TestPostNodeDumpImages_WhenSyncFails_ExpectOperationErrorOnSyncStage(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "article", &mockGitCommitter{syncErr: errors.New("sync failed")})

	code, start := doDumpJSON(t, mux, http.MethodPost, "/api/nodes/topic/my-node/dump-images")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		stCode, st := doDumpJSON(t, mux, http.MethodGet, "/api/node-dump-images/"+start.ID)

		return stCode == http.StatusOK && st.Status == "error" && st.Stage == "sync"
	}, time.Second, 20*time.Millisecond)
}

func TestGetNodeDumpImagesStatus_WhenOperationMissing_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "article", &mockGitCommitter{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-dump-images/missing-op", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetNodeDumpImagesLogs_WhenAfterInvalid_Expect400(t *testing.T) {
	t.Parallel()
	mux := setupDumpImagesHandler(t, "article", &mockGitCommitter{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-dump-images/some-id/logs?after=invalid", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
