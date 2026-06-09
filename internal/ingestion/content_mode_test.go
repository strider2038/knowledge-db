package ingestion_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestParseContentMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw    string
		mode   ingestion.ContentMode
		valid  bool
	}{
		{"", ingestion.ContentModeAuto, true},
		{"auto", ingestion.ContentModeAuto, true},
		{"verbatim", ingestion.ContentModeVerbatim, true},
		{"full_fetch", ingestion.ContentModeFullFetch, true},
		{"digest", ingestion.ContentModeDigest, true},
		{"link_bookmark", ingestion.ContentModeLinkBookmark, true},
		{"article", "", false},
	}

	for _, tt := range tests {
		mode, ok := ingestion.ParseContentMode(tt.raw)
		assert.Equal(t, tt.valid, ok, tt.raw)
		if tt.valid {
			assert.Equal(t, tt.mode, mode)
		}
	}
}

func TestResolveContentMode_ExplicitOverride(t *testing.T) {
	t.Parallel()

	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent:  "https://example.com",
		ContentMode: ingestion.ContentModeVerbatim,
	})

	assert.Equal(t, ingestion.ContentModeVerbatim, mode)
}

func TestResolveContentMode_PasteArticleWithURL_ExpectVerbatim(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("transcript paragraph. ", 50) + "https://www.youtube.com/watch?v=abc"
	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: body,
		TypeHint:   "article",
	})

	assert.Equal(t, ingestion.ContentModeVerbatim, mode)
}

func TestResolveContentMode_PasteArticleWithURL_AutoMode_ExpectVerbatim(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("Hermes desktop talk transcript paragraph. ", 40) + "https://www.youtube.com/watch?v=EJm8Ka-gVOc"
	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: body,
	})

	assert.Equal(t, ingestion.ContentModeVerbatim, mode)
}

func TestResolveContentMode_TelegramLongForm_ExpectVerbatim(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("paragraph about Go runtime. ", 30) + "https://github.com/example/repo"
	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: content,
		SourceURL:  "https://t.me/goproglib/1234",
		Profile: ingestion.SourceProfile{
			SourceKind:      kb.SourceKindArticle,
			ContentProfile:  kb.ContentProfileConceptualDigest,
			RecommendedType: "note",
		},
	})

	assert.Equal(t, ingestion.ContentModeVerbatim, mode)
}

func TestResolveContentMode_URLOnlyArticle_ExpectFullFetch(t *testing.T) {
	t.Parallel()

	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: "https://habr.com/ru/articles/123/",
		TypeHint:   "article",
	})

	assert.Equal(t, ingestion.ContentModeFullFetch, mode)
}

func TestResolveContentMode_URLOnlyProfile_ExpectDigest(t *testing.T) {
	t.Parallel()

	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: "https://github.com/pior/runnable",
		Profile: ingestion.SourceProfile{
			SourceKind:      kb.SourceKindRepository,
			ContentProfile:  kb.ContentProfileRepository,
			RecommendedType: "link",
		},
	})

	assert.Equal(t, ingestion.ContentModeDigest, mode)
}

func TestResolveContentMode_URLOnlyUnknown_ExpectLinkBookmark(t *testing.T) {
	t.Parallel()

	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: "https://example.invalid/page",
		Profile: ingestion.SourceProfile{
			SourceKind:      kb.SourceKindUnknown,
			ContentProfile:  kb.ContentProfileLinkBookmark,
			RecommendedType: "link",
		},
	})

	assert.Equal(t, ingestion.ContentModeLinkBookmark, mode)
}

func TestResolveContentMode_TextMarkers(t *testing.T) {
	t.Parallel()

	mode := ingestion.ResolveContentMode(ingestion.ResolveInput{
		RawContent: "Сохрани полную статью https://example.com/post",
	})

	assert.Equal(t, ingestion.ContentModeFullFetch, mode)
}

func TestResolveRefreshContentMode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ingestion.ContentModeFullFetch, ingestion.ResolveRefreshContentMode("article", "", "https://example.com", ""))
	assert.Equal(t, ingestion.ContentModeDigest, ingestion.ResolveRefreshContentMode("link", "repository_profile", "https://github.com/a/b", ""))
	assert.Equal(t, ingestion.ContentModeLinkBookmark, ingestion.ResolveRefreshContentMode("link", "link_bookmark", "https://t.me/channel", ""))
	assert.Equal(t, ingestion.ContentModeDigest, ingestion.ResolveRefreshContentMode("note", "conceptual_digest", "https://example.com", ""))
	assert.Equal(t, ingestion.ContentModeVerbatim, ingestion.ResolveRefreshContentMode("note", "", "", "existing body"))
}
