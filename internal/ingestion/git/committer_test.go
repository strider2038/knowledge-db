package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gittool "github.com/strider2038/knowledge-db/internal/ingestion/git"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	gitRun("init")
	gitRun("config", "user.email", "test@test.com")
	gitRun("config", "user.name", "Test")

	readmePath := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("# test\n"), 0o644))
	gitRun("add", "README.md")
	gitRun("commit", "-m", "init")

	return dir
}

func TestCommitNode_WhenNewFile_ExpectCommitCreated(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	ctx := context.Background()

	nodePath := filepath.Join(dir, "go", "concurrency", "goroutine-leak")
	require.NoError(t, os.MkdirAll(nodePath, 0o755))
	mdPath := filepath.Join(nodePath, "goroutine-leak.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("---\nkeywords: [goroutines]\n---\n"), 0o644))

	committer := gittool.NewExecGitCommitter(dir)
	err := committer.CommitNode(ctx, mdPath, "add: goroutine-leak")

	require.NoError(t, err)

	cmd := exec.CommandContext(ctx, "git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "goroutine-leak")
}

func TestSync_WhenNoRemote_ExpectError(t *testing.T) {
	t.Parallel()
	dir := initGitRepo(t)
	ctx := context.Background()

	committer := gittool.NewExecGitCommitter(dir)
	err := committer.Sync(ctx)

	assert.Error(t, err)
}
