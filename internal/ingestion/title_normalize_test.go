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

func TestNormalizeTitle_WhenTelegramPostTitle_ExpectCleanTitleWithTrailingEmoji(t *testing.T) {
	t.Parallel()

	title := normalizeTitle("🌐 **Пакет из стандартной библиотеки Go, про который почти никто не знает**")
	assert.Equal(t, "Пакет из стандартной библиотеки Go, про который почти никто не знает 🌐", title)
}
