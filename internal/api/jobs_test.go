package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

func setupJobsHandler(t *testing.T) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	nodeContent := `---
type: article
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
source_url: "https://example.com"
---

Content`
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(nodeContent), 0o644))
	h := api.NewHandler(tmp, &mockRefreshIngester{
		node: nil,
		err:  ingestion.ErrNotImplemented,
	})
	h.SetNodeNormalizer(&mockNodeNormalizer{})
	h.SetGitCommitter(&mockGitCommitter{}, nil, false)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func TestJobsAPI_WhenStartNormalize_ExpectStatusAndLogs(t *testing.T) {
	t.Parallel()
	mux := setupJobsHandler(t)
	body := `{"type":"node_normalize","target":"topic/my-node"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/jobs", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var started struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &started))
	require.NotEmpty(t, started.ID)

	require.Eventually(t, func() bool {
		stReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/jobs/"+started.ID, nil)
		stRec := httptest.NewRecorder()
		mux.ServeHTTP(stRec, stReq)
		if stRec.Code != http.StatusOK {
			return false
		}
		var status struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(stRec.Body.Bytes(), &status)

		return status.Status == "success"
	}, time.Second, 20*time.Millisecond)

	logReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/jobs/"+started.ID+"/logs", nil)
	logRec := httptest.NewRecorder()
	mux.ServeHTTP(logRec, logReq)
	require.Equal(t, http.StatusOK, logRec.Code)
	require.Contains(t, logRec.Body.String(), "normalization started")
}

func bytesReader(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}
