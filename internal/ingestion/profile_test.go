package ingestion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestClassifySource_WhenGitHubRepository_ExpectRepositoryProfile(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://github.com/pior/runnable", "", "", "")

	assert.Equal(t, kb.SourceKindRepository, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileRepository, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenDocsURL_ExpectDocumentationProfile(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://docs.github.com/en/actions", "GitHub Actions docs", "", "")

	assert.Equal(t, kb.SourceKindDocumentation, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileDocumentation, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenProductPage_ExpectProductProfile(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://example.com/pricing", "Example SaaS platform", "workflow integrations", "")

	assert.Equal(t, kb.SourceKindProductService, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileProduct, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenOnlineTool_ExpectOnlineToolProfile(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://example.com/json-validator", "JSON validator tool", "", "")

	assert.Equal(t, kb.SourceKindOnlineTool, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileOnlineTool, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenDirectory_ExpectDirectoryProfile(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://example.com/awesome-go", "Awesome Go", "curated list of libraries", "")

	assert.Equal(t, kb.SourceKindDirectoryCatalog, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileDirectory, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenLearningResource_ExpectLearningResourceProfile(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://example.com/go-course", "Go course", "", "")

	assert.Equal(t, kb.SourceKindLearningResource, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileLearningResource, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenArticle_ExpectConceptualDigest(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://example.com/blog/designing-go-libraries", "Designing Go Libraries", "article", "")

	assert.Equal(t, kb.SourceKindArticle, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileConceptualDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenNews_ExpectBriefDigest(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://techcrunch.com/2026/ai-model-release", "Model released", "", "")

	assert.Equal(t, kb.SourceKindNews, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileBriefDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenSocialPost_ExpectBriefDigest(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://x.com/openai/status/1", "", "", "")

	assert.Equal(t, kb.SourceKindSocialPost, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileBriefDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenUnknown_ExpectLinkBookmark(t *testing.T) {
	t.Parallel()

	profile := ClassifySource("https://example.invalid", "", "", "")

	assert.Equal(t, kb.SourceKindUnknown, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileLinkBookmark, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenTelegramLongFormMessage_ExpectConceptualDigestNote(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("Это длинный связный текст про Go и практику без cgo. ", 140)
	profile := ClassifySource("https://t.me/goproglib/1234", "Вызов C-функций из Go без Cgo", content, "")

	assert.Equal(t, kb.SourceKindArticle, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileConceptualDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenLongTechnicalTelegramPostWithLinks_ExpectNoteNotProduct(t *testing.T) {
	t.Parallel()

	content := `⭐️ Вызов C-функций из Go без Cgo

Если вы работали с Go и вам нужно было вызвать C-библиотеку, то вы наверняка сталкивались с Cgo.

Cgo работает, но тянет за собой целый набор проблем. Нужен C-компилятор на каждой целевой платформе.

purego решает всё это, позволяя вызывать C-функции из чистого Go.

Без Cgo отпадает необходимость в C-компиляторе. Вы можете собирать проект под другую платформу.

Работает и при CGO_ENABLED=1. Это значит, что можно портировать проект с Cgo на purego постепенно.

Типичные сценарии — работа с системными библиотеками, графические движки, аудио, нативные SDK.

➡️ Репозиторий https://clc.to/jjgR-g
📍 Навигация: https://clc.to/fuWG8g https://clc.to/_Jfhrg https://clc.to/zbSIdg
🐸 Библиотека Go-разработчика http://t.me/goproglib`

	profile := ClassifySource("https://t.me/goproglib/1234", "Вызов C-функций из Go без Cgo", content, "")

	assert.Equal(t, kb.SourceKindArticle, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileConceptualDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}
