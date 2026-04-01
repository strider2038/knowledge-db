package urlutil //nolint:testpackage // need access to stripTrackingParams

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeURL_Empty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	out, err := NormalizeURL(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestNormalizeURL_NonHTTP(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []string{"ftp://example.com", "mailto:foo@bar.com", "javascript:void(0)"}
	for _, u := range tests {
		out, err := NormalizeURL(ctx, u)
		require.NoError(t, err)
		assert.Equal(t, u, out)
	}
}

func TestNormalizeURL_StripUTM(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	in := srv.URL + "?utm_source=foo&utm_medium=bar&baz=qux"
	out, err := NormalizeURL(ctx, in)
	require.NoError(t, err)
	assert.Contains(t, out, "baz=qux")
	assert.NotContains(t, out, "utm_source")
	assert.NotContains(t, out, "utm_medium")
}

func TestNormalizeURL_StripUTM_OnlyUTM(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	in := srv.URL + "?utm_source=foo&utm_campaign=bar"
	out, err := NormalizeURL(ctx, in)
	require.NoError(t, err)
	assert.True(t, out == srv.URL || out == srv.URL+"/", "got %q", out)
}

func TestNormalizeURL_HTMLRedirectWindowLocation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	short := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html><script>window.location.href = "%s"</script>`, final.URL)
	}))
	defer short.Close()

	out, err := NormalizeURL(ctx, short.URL)
	require.NoError(t, err)
	assert.Contains(t, out, final.URL)
}

func TestNormalizeURL_Redirect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	shortener := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer shortener.Close()

	out, err := NormalizeURL(ctx, shortener.URL)
	require.NoError(t, err)
	assert.True(t, out == final.URL || out == final.URL+"/", "got %q, expected final URL", out)
}

func TestNormalizeURL_TMe_NoRedirect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	in := "https://t.me/channel/123?utm_source=foo"
	out, err := NormalizeURL(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, "https://t.me/channel/123", out)
}

func TestNormalizeURL_TelegramOrg_NoRedirect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	in := "https://telegram.org/something?utm_medium=bar"
	out, err := NormalizeURL(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, "https://telegram.org/something", out)
}

func TestStripTrackingParamsFromURL_GitHubRepo(t *testing.T) {
	t.Parallel()
	in := "https://github.com/SunWeb3Sec/llm-sast-scanner?utm_source=foo"
	out := StripTrackingParamsFromURL(in)
	assert.Equal(t, "https://github.com/SunWeb3Sec/llm-sast-scanner", out)
}

func TestStripTrackingParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"https://example.com?utm_source=foo&bar=baz", "https://example.com?bar=baz"},
		{"https://example.com?bar=baz", "https://example.com?bar=baz"},
		{"https://example.com", "https://example.com"},
		{"https://example.com?utm_source=a&utm_medium=b", "https://example.com"},
		{"https://example.com/page?fbclid=xx&keep=1", "https://example.com/page?keep=1"},
		{"https://example.com?gclid=abc&x=y", "https://example.com?x=y"},
	}
	for _, tt := range tests {
		got := stripTrackingParams(tt.in)
		assert.Equal(t, tt.want, got)
	}
}
