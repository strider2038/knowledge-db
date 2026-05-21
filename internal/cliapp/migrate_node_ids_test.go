package cliapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeNodeWithoutID(t *testing.T, base, relPath string) {
	t.Helper()
	dir := filepath.Join(base, filepath.Dir(relPath))
	require.NoError(t, os.MkdirAll(dir, 0o755))
	content := `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Migrate Me
---

# Body`
	require.NoError(t, os.WriteFile(filepath.Join(base, relPath), []byte(content), 0o644))
}

func TestMigrateNodeIDs_WhenDryRun_ExpectNoFileChange(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeNodeWithoutID(t, base, "topic/note.md")

	app := New()
	err := app.Run(context.Background(), []string{"kb", "migrate-node-ids", "--path", base, "--dry-run"})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(base, "topic/note.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "\nid:")
}

func TestMigrateNodeIDs_WhenApplyThenRerun_ExpectIdempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeNodeWithoutID(t, base, "topic/note.md")

	app := New()
	err := app.Run(context.Background(), []string{"kb", "migrate-node-ids", "--path", base})
	require.NoError(t, err)

	afterFirst, err := os.ReadFile(filepath.Join(base, "topic/note.md"))
	require.NoError(t, err)
	require.Contains(t, string(afterFirst), "\nid:")

	err = app.Run(context.Background(), []string{"kb", "migrate-node-ids", "--path", base})
	require.NoError(t, err)

	afterSecond, err := os.ReadFile(filepath.Join(base, "topic/note.md"))
	require.NoError(t, err)
	assert.Equal(t, string(afterFirst), string(afterSecond))
}
