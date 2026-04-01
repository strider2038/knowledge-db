package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
)

func TestPickResourceURLFromMessageText_MarkdownLinkFromTelegramEntity(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Как после entitiesToMarkdown для text_link: [видимый текст](реальный URL)
	text := `[исходники](https://github.com/SunWeb3Sec/llm-sast-scanner)`

	out := llm.PickResourceURLFromMessageText(ctx, text)
	assert.Equal(t, "https://github.com/SunWeb3Sec/llm-sast-scanner", out)
}

func TestPickResourceURLFromMessageText_GitHubRepoInParentheses(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	text := `Telegram-канал (откуда получен контент, НЕ является source_url ресурса): https://t.me/vibe_coding, автор: X

llm-sast-scanner: описание.

исходники (https://github.com/SunWeb3Sec/llm-sast-scanner)`

	out := llm.PickResourceURLFromMessageText(ctx, text)
	assert.Equal(t, "https://github.com/SunWeb3Sec/llm-sast-scanner", out)
}

func TestPickResourceURLFromMessageText_PrefersRepoOverDocsGitHub(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	text := `См. https://docs.github.com/en и репо https://github.com/org/repo`

	out := llm.PickResourceURLFromMessageText(ctx, text)
	assert.Equal(t, "https://github.com/org/repo", out)
}

func TestPickResourceURLFromMessageText_SkipsTelegram(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	text := `Канал https://t.me/foo и ссылка https://example.com/tool`

	out := llm.PickResourceURLFromMessageText(ctx, text)
	assert.Equal(t, "https://example.com/tool", out)
}

func TestPickResourceURLFromMessageText_Empty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assert.Empty(t, llm.PickResourceURLFromMessageText(ctx, "только текст без ссылок"))
}
