package kb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandURLsInString_TelegramStripUTM(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	in := "---\nsource_url: https://t.me/goproglib?utm_source=x\n---\n\n[Hi](https://t.me/y?utm_medium=z)\n"

	out, res := kb.ExpandURLsInString(ctx, in)

	assert.True(t, res.Changed)
	assert.NotContains(t, out, "utm_")
	assert.Contains(t, out, "https://t.me/goproglib")
	assert.Contains(t, out, "https://t.me/y")
	assert.Empty(t, res.FailedURLs)
}

func TestExpandURLsInString_Autolink(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	in := "See <https://t.me/x?utm_source=a>\n"

	out, res := kb.ExpandURLsInString(ctx, in)

	assert.True(t, res.Changed)
	assert.Contains(t, out, "<https://t.me/x>")
	assert.NotContains(t, out, "utm_")
	assert.Empty(t, res.FailedURLs)
}

func TestExpandURLsInString_RedirectShortener(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	short := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer short.Close()

	in := "[link](" + short.URL + ")\n"
	out, res := kb.ExpandURLsInString(ctx, in)

	assert.True(t, res.Changed)
	assert.Contains(t, out, final.URL)
	assert.Empty(t, res.FailedURLs)
}

func TestWriteExpandURLsFile_DryRunNoWrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	orig := "---\nx: 1\n---\n\n[U](https://t.me/z?utm_source=1)\n"
	require.NoError(t, os.WriteFile(path, []byte(orig), 0o644))

	res, err := kb.WriteExpandURLsFile(ctx, path, true)
	require.NoError(t, err)
	assert.True(t, res.Changed)

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, orig, string(after))
}
