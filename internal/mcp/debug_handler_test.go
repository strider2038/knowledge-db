package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/debugdata"
)

type debugStoreStub struct{}

func (s *debugStoreStub) ListIssues(_ context.Context, _ int) ([]debugdata.Issue, error) {
	return []debugdata.Issue{{
		ID:        "issue-1",
		Status:    debugdata.IssueStatusNew,
		Title:     "Bug",
		Page:      "node",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}}, nil
}

func (s *debugStoreStub) ReadIssue(_ context.Context, issueID string) (debugdata.Issue, error) {
	return debugdata.Issue{ID: issueID, Status: debugdata.IssueStatusNew, Title: "Bug"}, nil
}

func (s *debugStoreStub) UpdateIssueStatus(_ context.Context, issueID, status string) (debugdata.Issue, error) {
	now := time.Now().UTC()

	return debugdata.Issue{
		ID:        issueID,
		Status:    status,
		Title:     "Bug",
		Page:      "node",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *debugStoreStub) ReadLastTelegramRaw(_ context.Context, _ int) ([]debugdata.TelegramRawRecord, error) {
	return []debugdata.TelegramRawRecord{{UpdateID: 1, ReceivedAt: time.Now().UTC()}}, nil
}

func TestDebugHandler_WhenInvalidKey_Expect401(t *testing.T) {
	t.Parallel()
	h := NewDebugHandler("debug-key", &debugStoreStub{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/mcp/debug", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDebugHandler_WhenValidKey_ExpectAccessible(t *testing.T) {
	t.Parallel()
	h := NewDebugHandler("debug-key", &debugStoreStub{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/mcp/debug", strings.NewReader(`{"jsonrpc":"2.0","id":1}`))
	req.Header.Set("Authorization", "Bearer debug-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestDebugHandler_UpdateIssueStatusTool_WhenValidInput_ExpectAccessible(t *testing.T) {
	t.Parallel()
	h := NewDebugHandler("debug-key", &debugStoreStub{})
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"debug_update_issue_status","arguments":{"id":"issue-1","status":"fixed"}}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/mcp/debug", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer debug-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
}
