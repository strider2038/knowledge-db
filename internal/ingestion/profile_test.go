package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestClassifySource_WhenGitHubRepository_ExpectRepositoryProfile(t *testing.T) {
	profile := ClassifySource("https://github.com/pior/runnable", "", "", "")

	assert.Equal(t, kb.SourceKindRepository, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileRepository, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenDocsURL_ExpectDocumentationProfile(t *testing.T) {
	profile := ClassifySource("https://docs.github.com/en/actions", "GitHub Actions docs", "", "")

	assert.Equal(t, kb.SourceKindDocumentation, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileDocumentation, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenProductPage_ExpectProductProfile(t *testing.T) {
	profile := ClassifySource("https://example.com/pricing", "Example SaaS platform", "workflow integrations", "")

	assert.Equal(t, kb.SourceKindProductService, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileProduct, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenOnlineTool_ExpectOnlineToolProfile(t *testing.T) {
	profile := ClassifySource("https://example.com/json-validator", "JSON validator tool", "", "")

	assert.Equal(t, kb.SourceKindOnlineTool, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileOnlineTool, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenDirectory_ExpectDirectoryProfile(t *testing.T) {
	profile := ClassifySource("https://example.com/awesome-go", "Awesome Go", "curated list of libraries", "")

	assert.Equal(t, kb.SourceKindDirectoryCatalog, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileDirectory, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenLearningResource_ExpectLearningResourceProfile(t *testing.T) {
	profile := ClassifySource("https://example.com/go-course", "Go course", "", "")

	assert.Equal(t, kb.SourceKindLearningResource, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileLearningResource, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}

func TestClassifySource_WhenArticle_ExpectConceptualDigest(t *testing.T) {
	profile := ClassifySource("https://example.com/blog/designing-go-libraries", "Designing Go Libraries", "article", "")

	assert.Equal(t, kb.SourceKindArticle, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileConceptualDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenNews_ExpectBriefDigest(t *testing.T) {
	profile := ClassifySource("https://techcrunch.com/2026/ai-model-release", "Model released", "", "")

	assert.Equal(t, kb.SourceKindNews, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileBriefDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenSocialPost_ExpectBriefDigest(t *testing.T) {
	profile := ClassifySource("https://x.com/openai/status/1", "", "", "")

	assert.Equal(t, kb.SourceKindSocialPost, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileBriefDigest, profile.ContentProfile)
	assert.Equal(t, "note", profile.RecommendedType)
}

func TestClassifySource_WhenUnknown_ExpectLinkBookmark(t *testing.T) {
	profile := ClassifySource("https://example.invalid", "", "", "")

	assert.Equal(t, kb.SourceKindUnknown, profile.SourceKind)
	assert.Equal(t, kb.ContentProfileLinkBookmark, profile.ContentProfile)
	assert.Equal(t, "link", profile.RecommendedType)
}
