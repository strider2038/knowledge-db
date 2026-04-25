package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	igit "github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/api"
)

// mockGitCommitter implements GitCommitter for testing.
type mockGitCommitter struct {
	status    *igit.GitStatus
	statusErr error
	commitErr error
	diffStat  string
	diffErr   error
	lastMsg   string
}

func (m *mockGitCommitter) CommitNode(_ context.Context, _, _ string) error { return nil }
func (m *mockGitCommitter) Sync(_ context.Context) error                    { return nil }
func (m *mockGitCommitter) Status(_ context.Context) (*igit.GitStatus, error) {
	return m.status, m.statusErr
}
func (m *mockGitCommitter) CommitAll(_ context.Context, message string) error {
	m.lastMsg = message

	return m.commitErr
}
func (m *mockGitCommitter) DiffStat(_ context.Context) (string, error) {
	return m.diffStat, m.diffErr
}

func setupGitHandler(t *testing.T, committer igit.GitCommitter, gitDisabled bool) http.Handler {
	t.Helper()
	h := api.NewHandler(t.TempDir(), &ingestion.StubIngester{})
	h.SetGitCommitter(committer, nil, gitDisabled)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func TestGetGitStatus_WhenChanges_ExpectOK(t *testing.T) {
	t.Parallel()
	mux := setupGitHandler(t, &mockGitCommitter{
		status: &igit.GitStatus{HasChanges: true, ChangedFiles: 3},
	}, false)

	resp := apitest.HandleGET(t, mux, "/api/git/status")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("has_changes").IsTrue()
		json.Node("changed_files").IsInteger().EqualTo(3)
	})
}

func TestGetGitStatus_WhenNoChanges_ExpectOK(t *testing.T) {
	t.Parallel()
	mux := setupGitHandler(t, &mockGitCommitter{
		status: &igit.GitStatus{HasChanges: false, ChangedFiles: 0},
	}, false)

	resp := apitest.HandleGET(t, mux, "/api/git/status")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("has_changes").IsFalse()
		json.Node("changed_files").IsInteger().EqualTo(0)
	})
}

func TestGetGitStatus_WhenGitDisabled_Expect503(t *testing.T) {
	t.Parallel()
	mux := setupGitHandler(t, &mockGitCommitter{}, true)

	resp := apitest.HandleGET(t, mux, "/api/git/status")

	resp.HasCode(503)
}

func TestPostGitCommit_WhenEmptyJSONBody_ExpectOK(t *testing.T) {
	t.Parallel()
	mc := &mockGitCommitter{
		status:   &igit.GitStatus{HasChanges: true, ChangedFiles: 1},
		diffStat: "a.md | 1 +\n",
	}
	mux := setupGitHandler(t, mc, false)

	resp := apitest.HandlePOST(t, mux, "/api/git/commit",
		strings.NewReader(""),
		apitest.WithJSONContentType())

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("committed").IsTrue()
		json.Node("message").IsString()
	})
}

func TestPostGitCommit_WhenChanges_ExpectOK(t *testing.T) {
	t.Parallel()
	mc := &mockGitCommitter{
		status:  &igit.GitStatus{HasChanges: true, ChangedFiles: 2},
		diffStat: "file1.md | 10 +++\nfile2.md | 5 --\n",
	}
	mux := setupGitHandler(t, mc, false)

	resp := apitest.HandlePOST(t, mux, "/api/git/commit",
		strings.NewReader(`{}`),
		apitest.WithJSONContentType())

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("committed").IsTrue()
		json.Node("message").IsString()
	})
}

func TestPostGitCommit_WhenManualMessage_ExpectUsed(t *testing.T) {
	t.Parallel()
	mc := &mockGitCommitter{
		status: &igit.GitStatus{HasChanges: true, ChangedFiles: 1},
	}
	mux := setupGitHandler(t, mc, false)

	resp := apitest.HandlePOST(t, mux, "/api/git/commit",
		strings.NewReader(`{"message":"fix: update docs"}`),
		apitest.WithJSONContentType())

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("committed").IsTrue()
		json.Node("message").IsString().EqualTo("fix: update docs")
	})
	require.Equal(t, "fix: update docs", mc.lastMsg)
}

func TestPostGitCommit_WhenNoChanges_ExpectNotCommitted(t *testing.T) {
	t.Parallel()
	mc := &mockGitCommitter{
		status: &igit.GitStatus{HasChanges: false, ChangedFiles: 0},
	}
	mux := setupGitHandler(t, mc, false)

	resp := apitest.HandlePOST(t, mux, "/api/git/commit",
		strings.NewReader(`{}`),
		apitest.WithJSONContentType())

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("committed").IsFalse()
		json.Node("message").IsString().EqualTo("no changes to commit")
	})
}

func TestPostGitCommit_WhenGitDisabled_Expect503(t *testing.T) {
	t.Parallel()
	mux := setupGitHandler(t, &mockGitCommitter{}, true)

	resp := apitest.HandlePOST(t, mux, "/api/git/commit",
		strings.NewReader(`{}`),
		apitest.WithJSONContentType())

	resp.HasCode(503)
}
