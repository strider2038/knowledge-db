package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeTitle_StripsMarkdownAndMovesEmoji(t *testing.T) {
	t.Parallel()

	title := normalizeTitle("🔥 [httptrace](https://example.com/trace)")
	assert.Equal(t, "httptrace 🔥", title)
}
