package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/debugdata"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

type debugIssueStoreStub struct {
	lastPayload debugdata.IssuePayload
	lastID      string
	lastStatus  string
}

func (s *debugIssueStoreStub) WriteIssue(_ context.Context, payload debugdata.IssuePayload) (debugdata.Issue, error) {
	s.lastPayload = payload

	return debugdata.Issue{
		ID:        "issue-1",
		Status:    debugdata.IssueStatusNew,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *debugIssueStoreStub) UpdateIssueStatus(_ context.Context, issueID, status string) (debugdata.Issue, error) {
	s.lastID = issueID
	s.lastStatus = status

	return debugdata.Issue{
		ID:        issueID,
		Status:    status,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func TestPostDebugIssue_WhenValidPayload_ExpectOK(t *testing.T) {
	t.Parallel()
	h := api.NewHandler(t.TempDir(), &ingestion.StubIngester{})
	store := &debugIssueStoreStub{}
	h.SetDebugIssueStore(store)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/debug/issues", bytes.NewBufferString(`{
		"title":"UI bug",
		"description":"button is broken",
		"page":"node",
		"context":{"path":"a/b"}
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "UI bug", store.lastPayload.Title)
	require.Equal(t, "button is broken", store.lastPayload.Description)
}

func TestPatchDebugIssueStatus_WhenValidStatus_ExpectOK(t *testing.T) {
	t.Parallel()
	h := api.NewHandler(t.TempDir(), &ingestion.StubIngester{})
	store := &debugIssueStoreStub{}
	h.SetDebugIssueStore(store)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/debug/issues/issue-1", bytes.NewBufferString(`{"status":"fixed"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "issue-1", store.lastID)
	require.Equal(t, "fixed", store.lastStatus)
}

func TestPatchDebugIssueStatus_WhenInvalidStatus_ExpectBadRequest(t *testing.T) {
	t.Parallel()
	h := api.NewHandler(t.TempDir(), &ingestion.StubIngester{})
	h.SetDebugIssueStore(&debugIssueStoreStub{})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/debug/issues/issue-1", bytes.NewBufferString(`{"status":"unknown"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
