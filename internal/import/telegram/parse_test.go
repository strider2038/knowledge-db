package telegram_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/import/telegram"
)

func TestParseChat_TextAsString(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{
				"id": 1,
				"type": "message",
				"date_unixtime": "1557861184",
				"from": "Alice",
				"text": "Simple plain text"
			}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, int64(1), items[0].ID)
	assert.Equal(t, "1557861184", items[0].DateUnixTime)
	assert.Equal(t, "Simple plain text", items[0].Text)
	assert.Equal(t, "Alice", items[0].SourceAuthor)
	assert.Empty(t, items[0].SourceURL)
}

func TestParseChat_TextAsArray_PlainAndLink(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{
				"id": 2,
				"type": "message",
				"date_unixtime": "1557861185",
				"from": "Bob",
				"text": [
					{"type": "link", "text": "https://example.com"},
					{"type": "plain", "text": " - comment"}
				]
			}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "[https://example.com](https://example.com) - comment", items[0].Text)
	assert.Equal(t, "https://example.com", items[0].SourceURL)
}

func TestParseChat_TextEntities_TextLink(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{
				"id": 3,
				"type": "message",
				"date_unixtime": "1557861186",
				"from": "Charlie",
				"text": "Check this",
				"text_entities": [
					{"type": "plain", "text": "Check "},
					{"type": "text_link", "text": "this", "href": "https://link.example"}
				]
			}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "Check [this](https://link.example)", items[0].Text)
	assert.Equal(t, "https://link.example", items[0].SourceURL)
}

func TestParseChat_SourceAuthorPriority_ForwardedOverSaved(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{
				"id": 4,
				"type": "message",
				"date_unixtime": "1557861187",
				"from": "DirectSender",
				"forwarded_from": "ForwardedAuthor",
				"saved_from": "SavedChat",
				"text": "Message"
			}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "ForwardedAuthor", items[0].SourceAuthor)
}

func TestParseChat_SourceAuthorPriority_SavedOverFrom(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{
				"id": 5,
				"type": "message",
				"date_unixtime": "1557861188",
				"from": "DirectSender",
				"saved_from": "SavedChat",
				"text": "Message"
			}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "SavedChat", items[0].SourceAuthor)
}

func TestParseChat_SortByDateDesc(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{"id": 1, "type": "message", "date_unixtime": "1557861180", "from": "A", "text": "Old"},
			{"id": 2, "type": "message", "date_unixtime": "1557861190", "from": "B", "text": "New"}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "New", items[0].Text)
	assert.Equal(t, "Old", items[1].Text)
}

func TestParseChat_SkipsServiceMessages(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{"id": 1, "type": "service", "date_unixtime": "1557861180", "text": "Service"},
			{"id": 2, "type": "message", "date_unixtime": "1557861181", "from": "A", "text": "Real"}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "Real", items[0].Text)
}

func TestParseChat_SkipsEmptyText(t *testing.T) {
	t.Parallel()
	json := `{
		"id": 123,
		"name": "Test",
		"type": "personal_chat",
		"messages": [
			{"id": 1, "type": "message", "date_unixtime": "1557861180", "from": "A", "text": ""},
			{"id": 2, "type": "message", "date_unixtime": "1557861181", "from": "B"}
		]
	}`

	items, err := telegram.ParseChat([]byte(json))
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestParseChat_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := telegram.ParseChat([]byte("not json"))
	require.Error(t, err)
}
