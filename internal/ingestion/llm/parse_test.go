package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCreateNodeArgs_WhenSourceDate_ExpectParsed(t *testing.T) {
	t.Parallel()

	args := `{"keywords":["go"],"annotation":"test","theme_path":"go","slug":"test","type":"note","title":"Test","source_date":"2026-01-15"}`
	result, err := parseCreateNodeArgs(args)

	require.NoError(t, err)
	require.NotNil(t, result.SourceDate)
	expected := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, *result.SourceDate)
}

func TestParseCreateNodeArgs_WhenSourceAuthor_ExpectParsed(t *testing.T) {
	t.Parallel()

	args := `{"keywords":["go"],"annotation":"test","theme_path":"go","slug":"test","type":"article","title":"Test","source_author":"Иван Петров"}`
	result, err := parseCreateNodeArgs(args)

	require.NoError(t, err)
	assert.Equal(t, "Иван Петров", result.SourceAuthor)
}

func TestParseCreateNodeArgs_WhenSourceProfile_ExpectParsed(t *testing.T) {
	t.Parallel()

	args := `{"keywords":["go"],"annotation":"test","theme_path":"go","slug":"test","type":"link","title":"Test","source_kind":"repository","content_profile":"repository_profile"}`
	result, err := parseCreateNodeArgs(args)

	require.NoError(t, err)
	assert.Equal(t, "repository", result.SourceKind)
	assert.Equal(t, "repository_profile", result.ContentProfile)
}
