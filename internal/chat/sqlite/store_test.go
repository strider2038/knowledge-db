package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_SummarizeAndTrim(t *testing.T) {
	t.Parallel()
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	store.maxMessages = 3

	ctx := context.Background()
	_, err = store.CreateSession(ctx, "s1", "Chat")
	require.NoError(t, err)

	for range 8 {
		require.NoError(t, store.AddMessage(ctx, "s1", "user", "message", false))
	}

	require.NoError(t, store.SummarizeAndTrim(ctx, "s1"))

	details, err := store.GetSession(ctx, "s1")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(details.Messages), 3)

	prompt, err := store.BuildPromptMessages(ctx, "s1")
	require.NoError(t, err)
	assert.NotEmpty(t, prompt)
}

func TestStore_SummarizeAndTrim_WhenRuneLimitExceeded_ExpectSummary(t *testing.T) {
	t.Parallel()
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	store.maxMessages = 10
	store.maxContextRunes = 450

	ctx := context.Background()
	_, err = store.CreateSession(ctx, "s1", "Chat")
	require.NoError(t, err)

	for _, message := range []string{
		"first long message " + strings.Repeat("a", 250),
		"second long message " + strings.Repeat("b", 250),
		"third recent message " + strings.Repeat("c", 20),
	} {
		require.NoError(t, store.AddMessage(ctx, "s1", "user", message, false))
	}

	require.NoError(t, store.SummarizeAndTrim(ctx, "s1"))

	details, err := store.GetSession(ctx, "s1")
	require.NoError(t, err)
	if assert.Len(t, details.Messages, 1) {
		assert.Contains(t, details.Messages[0].Content, "third recent message")
	}
	prompt, err := store.BuildPromptMessages(ctx, "s1")
	require.NoError(t, err)
	assert.Contains(t, prompt[0]["content"], "Summary of previous dialog")
}

func TestStore_AutoRenameFromFirstUserMessage(t *testing.T) {
	t.Parallel()
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "s1", "")
	require.NoError(t, err)
	require.Equal(t, "Новый чат", session.Title)

	require.NoError(t, store.AddMessage(ctx, "s1", "user", "   Как устроен sqlite wal mode в двух словах   ", false))

	details, err := store.GetSession(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "Как устроен sqlite wal mode в двух словах", details.Session.Title)
}

func TestStore_GetSessionEmptyMessagesIsNotNilSlice(t *testing.T) {
	t.Parallel()
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	_, err = store.CreateSession(ctx, "s1", "Chat")
	require.NoError(t, err)

	details, err := store.GetSession(ctx, "s1")
	require.NoError(t, err)
	assert.NotNil(t, details.Messages)
	assert.Empty(t, details.Messages)
}
