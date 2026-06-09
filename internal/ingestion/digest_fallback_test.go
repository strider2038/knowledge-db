package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
)

func TestBuildFallbackModeContent_WhenRepositoryDigest_ExpectStructuredBody(t *testing.T) {
	t.Parallel()

	body := buildFallbackModeContent(
		ContentModeDigest,
		&llm.ProcessResult{
			Title:          "Runnable",
			Annotation:     "Go library for graceful shutdown.",
			SourceURL:      "https://github.com/pior/runnable",
			ContentProfile: "repository_profile",
		},
		"",
	)

	assert.Contains(t, body, "## Назначение")
	assert.Contains(t, body, "Go library for graceful shutdown.")
	assert.Contains(t, body, "## Источник")
	assert.Contains(t, body, "https://github.com/pior/runnable")
}

func TestBuildFallbackModeContent_WhenLinkBookmark_ExpectCompactBody(t *testing.T) {
	t.Parallel()

	body := buildFallbackModeContent(
		ContentModeLinkBookmark,
		&llm.ProcessResult{
			Title:      "net/http",
			Annotation: "HTTP package in Go standard library.",
			SourceURL:  "https://pkg.go.dev/net/http",
		},
		"",
	)

	assert.Contains(t, body, "net/http — https://pkg.go.dev/net/http")
	assert.Contains(t, body, "HTTP package in Go standard library.")
}

func TestBuildFallbackModeContent_WhenNoFacts_ExpectEmpty(t *testing.T) {
	t.Parallel()

	body := buildFallbackModeContent(
		ContentModeDigest,
		&llm.ProcessResult{ContentProfile: "repository_profile"},
		"",
	)

	assert.Empty(t, body)
}
