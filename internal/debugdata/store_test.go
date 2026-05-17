package debugdata

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStore_WriteIssueAndUpdateStatus(t *testing.T) {
	t.Parallel()

	s := NewStore(t.TempDir())
	issue, err := s.WriteIssue(context.Background(), IssuePayload{
		Title:       "Broken UI",
		Description: "Button does not work",
		Page:        "node",
		Context:     map[string]any{"nodePath": "a/b"},
	})
	require.NoError(t, err)
	require.Equal(t, IssueStatusNew, issue.Status)

	updated, err := s.UpdateIssueStatus(context.Background(), issue.ID, IssueStatusFixed)
	require.NoError(t, err)
	require.Equal(t, IssueStatusFixed, updated.Status)
}

func TestStore_WhenWriteIssue_ExpectFileUnderDotKB(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	s := NewStore(root)
	issue, err := s.WriteIssue(context.Background(), IssuePayload{
		Title:       "Broken UI",
		Description: "Button does not work",
		Page:        "node",
		Context:     map[string]any{"nodePath": "a/b"},
	})
	require.NoError(t, err)

	var issuePath string
	_ = filepath.WalkDir(filepath.Join(root, ".kb"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		if strings.Contains(path, issue.ID+".md") {
			issuePath = path
		}

		return nil
	})
	require.NotEmpty(t, issuePath)
	require.Contains(t, issuePath, filepath.Join(".kb", "issues"))
}

func TestStore_TelegramRawAppendAndCleanup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	s := NewStore(root)
	now := time.Now().UTC()
	old := now.Add(-15 * 24 * time.Hour)

	require.NoError(t, s.AppendTelegramRaw(context.Background(), TelegramRawRecord{
		ReceivedAt: old,
		UpdateID:   1,
		Payload:    json.RawMessage(`{"a":1}`),
	}))
	require.NoError(t, s.AppendTelegramRaw(context.Background(), TelegramRawRecord{
		ReceivedAt: now,
		UpdateID:   2,
		Payload:    json.RawMessage(`{"b":2}`),
	}))

	require.NoError(t, s.CleanupTelegramRaw(context.Background(), now, 14*24*time.Hour))

	_, err := os.Stat(filepath.Join(root, ".kb", "telegram-raw", old.Format("2006-01-02")+".ndjson"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root, ".kb", "telegram-raw", now.Format("2006-01-02")+".ndjson"))
	require.NoError(t, err)
}
