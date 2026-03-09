package kb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestExtractImageURLs_WhenMarkdownImage_ExpectExtracted(t *testing.T) {
	t.Parallel()

	body := `![Diagram](https://example.com/diagram.png)`
	refs := kb.ExtractImageURLs(body)

	require.Len(t, refs, 1)
	assert.Equal(t, "https://example.com/diagram.png", refs[0].URL)
	assert.Equal(t, "Diagram", refs[0].Alt)
	assert.True(t, refs[0].IsImageSyntax)
}

func TestExtractImageURLs_WhenLinkWithImageExt_ExpectExtracted(t *testing.T) {
	t.Parallel()

	body := `[screenshot](https://x.com/img.jpg)`
	refs := kb.ExtractImageURLs(body)

	require.Len(t, refs, 1)
	assert.Equal(t, "https://x.com/img.jpg", refs[0].URL)
	assert.False(t, refs[0].IsImageSyntax)
}

func TestExtractImageURLs_WhenLocalPath_ExpectIgnored(t *testing.T) {
	t.Parallel()

	body := `![Local](images/foo.png) and ![Other](./bar.jpg)`
	refs := kb.ExtractImageURLs(body)

	assert.Empty(t, refs)
}

func TestExtractImageURLs_WhenMixed_ExpectOnlyRemote(t *testing.T) {
	t.Parallel()

	body := `![Local](images/local.png)
![Remote](https://cdn.example.com/photo.webp)`
	refs := kb.ExtractImageURLs(body)

	require.Len(t, refs, 1)
	assert.Equal(t, "https://cdn.example.com/photo.webp", refs[0].URL)
}

func TestExtractImageURLs_WhenDuplicateURL_ExpectDeduplicated(t *testing.T) {
	t.Parallel()

	body := `![A](https://example.com/same.png)
![B](https://example.com/same.png)`
	refs := kb.ExtractImageURLs(body)

	require.Len(t, refs, 1)
	assert.Equal(t, "https://example.com/same.png", refs[0].URL)
}

func TestImageFilename_WhenValidURL_ExpectHashAndExt(t *testing.T) {
	t.Parallel()

	fn, err := kb.ImageFilename("https://example.com/image.png")
	require.NoError(t, err)
	assert.Len(t, fn, 12+4) // 12 hex + ".png"
	assert.Contains(t, fn, ".png")
}

func TestImageFilename_WhenSameURL_ExpectSameFilename(t *testing.T) {
	t.Parallel()

	url := "https://example.com/photo.jpg"
	fn1, err1 := kb.ImageFilename(url)
	fn2, err2 := kb.ImageFilename(url)
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, fn1, fn2)
}

func TestImageFilename_WhenNoImageExt_ExpectError(t *testing.T) {
	t.Parallel()

	_, err := kb.ImageFilename("https://example.com/page.html")
	require.Error(t, err)
}

func TestRunDumpImages_WhenRemoteImages_ExpectDownloadedAndReplaced(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	mdContent := `---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

# Test

![Diagram](` + srv.URL + `/diagram.png)
`

	fs := afero.NewMemMapFs()
	basePath := "/"
	mdPath := filepath.Join(basePath, "topic", "article.md")
	_ = fs.MkdirAll(filepath.Dir(mdPath), 0o755)
	_ = afero.WriteFile(fs, mdPath, []byte(mdContent), 0o644)

	ctx := context.Background()
	client := srv.Client()

	modified, downloadErrors, results, err := kb.RunDumpImages(ctx, fs, client, basePath, "topic", "article", false, nil)
	require.NoError(t, err)
	assert.True(t, modified)
	assert.Empty(t, downloadErrors)
	assert.Nil(t, results)

	imagesDir := filepath.Join(basePath, "topic", "article", "images")
	entries, err := afero.ReadDir(fs, imagesDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Regexp(t, `^[a-f0-9]{12}\.png$`, entries[0].Name())

	data, err := afero.ReadFile(fs, mdPath)
	require.NoError(t, err)
	body := string(data)
	assert.Contains(t, body, "article/images/")
	assert.NotContains(t, body, srv.URL)
	assert.Contains(t, body, "updated:")
}

func TestRunDumpImages_WhenDryRun_ExpectNoFilesCreated(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	}))
	defer srv.Close()

	mdContent := `---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

![Img](` + srv.URL + `/x.png)
`

	fs := afero.NewMemMapFs()
	basePath := "/"
	mdPath := filepath.Join(basePath, "theme", "node.md")
	_ = fs.MkdirAll(filepath.Dir(mdPath), 0o755)
	_ = afero.WriteFile(fs, mdPath, []byte(mdContent), 0o644)

	ctx := context.Background()
	client := srv.Client()

	modified, downloadErrors, results, err := kb.RunDumpImages(ctx, fs, client, basePath, "theme", "node", true, nil)
	require.NoError(t, err)
	assert.False(t, modified)
	assert.Empty(t, downloadErrors)
	require.Len(t, results, 1)
	assert.Equal(t, srv.URL+"/x.png", results[0].URL)
	assert.Contains(t, results[0].TargetPath, "node/images/")
	assert.Contains(t, results[0].TargetPath, ".png")

	exists, _ := afero.Exists(fs, filepath.Join(basePath, "theme", "node", "images"))
	assert.False(t, exists)
}
