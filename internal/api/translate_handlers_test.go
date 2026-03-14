package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
)

func setupTranslateTestHandler(t *testing.T, withQueue bool) http.Handler {
	t.Helper()
	const articlePath = "go/test-article"
	tmp := t.TempDir()
	dir := filepath.Join(tmp, filepath.FromSlash(articlePath))
	_ = os.MkdirAll(filepath.Dir(dir), 0o755)
	articleContent := `---
keywords: [go]
type: article
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
title: "Test Article"
---

Content in English.`
	slug := filepath.Base(dir)
	_ = os.WriteFile(filepath.Join(filepath.Dir(dir), slug+".md"), []byte(articleContent), 0o644)

	var queue *translationqueue.Queue
	if withQueue {
		queue = translationqueue.New(10)
	}
	h := api.NewHandlerWithUploads(tmp, "", &ingestion.StubIngester{}, queue)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func TestPostArticleTranslate_WhenQueueNil_Expect503(t *testing.T) {
	t.Parallel()
	handler := setupTranslateTestHandler(t, false)

	resp := apitest.HandlePOST(t, handler, "/api/articles/translate/go/test-article", nil)

	resp.HasCode(http.StatusServiceUnavailable)
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("error").IsString().EqualTo("translation service unavailable")
	})
}

func TestGetArticleTranslate_WhenQueueNil_Expect503(t *testing.T) {
	t.Parallel()
	handler := setupTranslateTestHandler(t, false)

	resp := apitest.HandleGET(t, handler, "/api/articles/translate/go/test-article")

	resp.HasCode(http.StatusServiceUnavailable)
}

func TestPostArticleTranslate_WhenNodeNotFound_Expect404(t *testing.T) {
	t.Parallel()
	handler := setupTranslateTestHandler(t, true)

	resp := apitest.HandlePOST(t, handler, "/api/articles/translate/go/missing-article", nil)

	resp.HasCode(http.StatusNotFound)
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("error").IsString().EqualTo("node not found")
	})
}

func TestPostArticleTranslate_WhenNotArticle_Expect400(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	_ = os.MkdirAll(themeDir, 0o755)
	noteContent := `---
keywords: [a]
type: note
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
---

Content`
	_ = os.WriteFile(filepath.Join(themeDir, "node1.md"), []byte(noteContent), 0o644)
	queue := translationqueue.New(10)
	h := api.NewHandlerWithUploads(tmp, "", &ingestion.StubIngester{}, queue)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	resp := apitest.HandlePOST(t, mux, "/api/articles/translate/topic/node1", nil)

	resp.HasCode(http.StatusBadRequest)
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("error").IsString().EqualTo("node is not an article")
	})
}

func TestPostArticleTranslate_WhenNoTranslation_ExpectPending(t *testing.T) {
	t.Parallel()
	handler := setupTranslateTestHandler(t, true)

	resp := apitest.HandlePOST(t, handler, "/api/articles/translate/go/test-article", nil)

	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("status").IsString().EqualTo("pending")
	})
}

func TestPostArticleTranslate_WhenAlreadyPending_ExpectNoDuplicate(t *testing.T) {
	t.Parallel()
	handler := setupTranslateTestHandler(t, true)

	resp1 := apitest.HandlePOST(t, handler, "/api/articles/translate/go/test-article", nil)
	resp1.IsOK()
	resp1.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("status").IsString().EqualTo("pending")
	})

	resp2 := apitest.HandlePOST(t, handler, "/api/articles/translate/go/test-article", nil)
	resp2.IsOK()
	resp2.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("status").IsString().EqualTo("pending")
	})
}

func TestGetArticleTranslate_WhenNoTranslation_ExpectNoneOrPending(t *testing.T) {
	t.Parallel()
	handler := setupTranslateTestHandler(t, true)

	resp := apitest.HandleGET(t, handler, "/api/articles/translate/go/test-article")

	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("status").IsString()
	})
}

func TestGetArticleTranslate_WhenTranslationExists_ExpectDone(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "go")
	_ = os.MkdirAll(themeDir, 0o755)
	articleContent := `---
keywords: [go]
type: article
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
title: "Test Article"
---

Content`
	_ = os.WriteFile(filepath.Join(themeDir, "test-article.md"), []byte(articleContent), 0o644)
	translationContent := `---
translation_of: test-article
lang: ru
---

Перевод`
	_ = os.WriteFile(filepath.Join(themeDir, "test-article.ru.md"), []byte(translationContent), 0o644)
	queue := translationqueue.New(10)
	h := api.NewHandlerWithUploads(tmp, "", &ingestion.StubIngester{}, queue)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	resp := apitest.HandleGET(t, mux, "/api/articles/translate/go/test-article")

	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("status").IsString().EqualTo("done")
	})
}

func TestPostArticleTranslate_WhenTranslationExists_ExpectDone(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "go")
	_ = os.MkdirAll(themeDir, 0o755)
	articleContent := `---
keywords: [go]
type: article
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
title: "Test Article"
---

Content`
	_ = os.WriteFile(filepath.Join(themeDir, "test-article.md"), []byte(articleContent), 0o644)
	translationContent := `---
translation_of: test-article
lang: ru
---

Перевод`
	_ = os.WriteFile(filepath.Join(themeDir, "test-article.ru.md"), []byte(translationContent), 0o644)
	queue := translationqueue.New(10)
	h := api.NewHandlerWithUploads(tmp, "", &ingestion.StubIngester{}, queue)
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	resp := apitest.HandlePOST(t, mux, "/api/articles/translate/go/test-article", nil)

	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("status").IsString().EqualTo("done")
	})
}
