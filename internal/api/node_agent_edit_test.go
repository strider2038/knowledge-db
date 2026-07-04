package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/muonsoft/errors"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type mockNodeAgentEditor struct {
	err error
}

func (m *mockNodeAgentEditor) EditNode(_ context.Context, _ string, _ *kb.Node, _ string, onLog func(stream, text string)) error {
	onLog("stdout", "edit line one")
	onLog("stderr", "edit line two")

	return m.err
}

type blockingAgentEditor struct {
	release chan struct{}
}

func (m *blockingAgentEditor) EditNode(ctx context.Context, _ string, _ *kb.Node, _ string, onLog func(stream, text string)) error {
	onLog("stdout", "working")
	select {
	case <-m.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type agentEditResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Stage    string `json:"stage"`
	SyncDone bool   `json:"sync_done"`
	EditOK   bool   `json:"edit_ok"`
	Error    string `json:"error"`
}

type agentEditLogsResponse struct {
	Entries []struct {
		Offset int64  `json:"offset"`
		Stream string `json:"stream"`
		Text   string `json:"text"`
	} `json:"entries"`
	NextOffset int64 `json:"next_offset"`
}

func setupAgentEditHandler(t *testing.T, editor api.NodeAgentEditor, committer *mockGitCommitter) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := "---\nkeywords: [test]\n---\n\nContent"
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetNodeAgentEditor(editor)
	h.SetGitCommitter(committer, nil, false)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func doAgentEditJSON(t *testing.T, mux http.Handler, nodePath string, instruction string) (int, agentEditResponse) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"path": nodePath, "instruction": instruction})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/nodes/agent-edit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out agentEditResponse
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}

	return rec.Code, out
}

func TestPostNodeAgentEdit_WhenSuccess_ExpectOperationSuccess(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	code, start := doAgentEditJSON(t, mux, "topic/my-node", "add keywords")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)
	require.Equal(t, "running", start.Status)

	require.Eventually(t, func() bool {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/"+start.ID, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var st agentEditResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &st)

		return rec.Code == http.StatusOK && st.Status == "success" && st.SyncDone && st.EditOK
	}, time.Second, 20*time.Millisecond)

	logsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/"+start.ID+"/logs", nil)
	logsRec := httptest.NewRecorder()
	mux.ServeHTTP(logsRec, logsReq)
	require.Equal(t, http.StatusOK, logsRec.Code)
	require.Contains(t, logsRec.Body.String(), "edit line one")
}

func TestPostNodeAgentEdit_WhenNodeMissing_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	code, _ := doAgentEditJSON(t, mux, "topic/missing", "fix title")
	require.Equal(t, http.StatusNotFound, code)
}

func TestPostNodeAgentEdit_WhenEmptyInstruction_Expect400(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	code, _ := doAgentEditJSON(t, mux, "topic/my-node", "   ")
	require.Equal(t, http.StatusBadRequest, code)
}

func TestGetNodeAgentEditLogs_WhenAfterProvided_ExpectOnlyNewEntries(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	code, start := doAgentEditJSON(t, mux, "topic/my-node", "improve intro")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/"+start.ID, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var st agentEditResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &st)

		return rec.Code == http.StatusOK && st.Status == "success"
	}, time.Second, 20*time.Millisecond)

	reqAfter := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/"+start.ID+"/logs?after=2", nil)
	recAfter := httptest.NewRecorder()
	mux.ServeHTTP(recAfter, reqAfter)
	require.Equal(t, http.StatusOK, recAfter.Code)

	var filtered agentEditLogsResponse
	require.NoError(t, json.Unmarshal(recAfter.Body.Bytes(), &filtered))
	for _, entry := range filtered.Entries {
		require.Greater(t, entry.Offset, int64(2))
	}
}

func TestGetNodeAgentEditLogs_WhenUnknownOperation_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/missing/logs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostNodeAgentEdit_WhenEditorFails_ExpectOperationError(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{err: errors.New("edit failed")}, &mockGitCommitter{})

	code, start := doAgentEditJSON(t, mux, "topic/my-node", "break things")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/"+start.ID, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var st agentEditResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &st)

		return rec.Code == http.StatusOK && st.Status == "error" && st.Stage == "edit"
	}, time.Second, 20*time.Millisecond)
}

func TestPostNodeAgentEdit_WhenAlreadyRunning_Expect409(t *testing.T) {
	t.Parallel()
	editor := &blockingAgentEditor{release: make(chan struct{})}
	mux := setupAgentEditHandler(t, editor, &mockGitCommitter{})

	code1, start1 := doAgentEditJSON(t, mux, "topic/my-node", "first run")
	require.Equal(t, http.StatusOK, code1)
	require.NotEmpty(t, start1.ID)

	code2, _ := doAgentEditJSON(t, mux, "topic/my-node", "second run")
	require.Equal(t, http.StatusConflict, code2)

	close(editor.release)
}

func TestPostNodeAgentEdit_WhenNoRunner_Expect503(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	code, _ := doAgentEditJSON(t, mux, "topic/my-node", "test")
	require.Equal(t, http.StatusServiceUnavailable, code)
}

func TestGetNodeAgentEditStatus_WhenUnknownOperation_Expect404(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/missing", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPostNodeAgentEdit_WhenInstructionTooLong_Expect400(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{})

	longInstruction := strings.Repeat("a", 16*1024+1)
	code, _ := doAgentEditJSON(t, mux, "topic/my-node", longInstruction)
	require.Equal(t, http.StatusBadRequest, code)
}

func TestPostNodeAgentEdit_WhenSyncFails_ExpectEditOKWithoutSync(t *testing.T) {
	t.Parallel()
	mux := setupAgentEditHandler(t, &mockNodeAgentEditor{}, &mockGitCommitter{syncErr: errors.New("sync failed")})

	code, start := doAgentEditJSON(t, mux, "topic/my-node", "add keywords")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, start.ID)

	require.Eventually(t, func() bool {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/node-agent-edit/"+start.ID, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var st agentEditResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &st)

		return rec.Code == http.StatusOK && st.Status == "error" && st.Stage == "sync" && st.EditOK && !st.SyncDone
	}, time.Second, 20*time.Millisecond)
}

func TestPostNodeAgentEdit_WhenCursorAgentMissing_Expect503(t *testing.T) {
	t.Setenv("PATH", "")
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := "---\nkeywords: [test]\n---\n\nContent"
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetNodeAgentEditor(api.NewCursorNodeAgentEditor())
	h.SetGitCommitter(&mockGitCommitter{}, nil, false)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	body, err := json.Marshal(map[string]string{"path": "topic/my-node", "instruction": "test"})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/nodes/agent-edit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "cursor-agent not found in PATH")
}
