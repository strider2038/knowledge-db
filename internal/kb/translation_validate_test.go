package kb_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestValidate_WhenValidTranslation_ExpectNoViolations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fs := afero.NewMemMapFs()
	base := "/base"
	themeDir := filepath.Join(base, "theme")
	_ = fs.MkdirAll(themeDir, 0o755)

	originalContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
---
# Original

Content here.
`
	translationContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
translation_of: my-article
lang: ru
---
# Оригинал

Контент здесь.

[[my-article|Original]]
`
	_ = afero.WriteFile(fs, filepath.Join(themeDir, "my-article.md"), []byte(originalContent), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(themeDir, "my-article.ru.md"), []byte(translationContent), 0o644)

	// Add translations to original
	originalWithTranslations := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
translations: [my-article.ru]
---
# Original

Content here.

[[my-article.ru|Русский перевод]]
`
	_ = afero.WriteFile(fs, filepath.Join(themeDir, "my-article.md"), []byte(originalWithTranslations), 0o644)

	store := kb.NewStore(fs)
	violations, err := store.Validate(ctx, base)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestValidate_WhenTranslationMissingTranslationOf_ExpectViolation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fs := afero.NewMemMapFs()
	base := "/base"
	themeDir := filepath.Join(base, "theme")
	_ = fs.MkdirAll(themeDir, 0o755)

	originalContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
---
# Original
`
	translationContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
lang: ru
---
# Оригинал
`
	_ = afero.WriteFile(fs, filepath.Join(themeDir, "my-article.md"), []byte(originalContent), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(themeDir, "my-article.ru.md"), []byte(translationContent), 0o644)

	store := kb.NewStore(fs)
	violations, err := store.Validate(ctx, base)

	require.NoError(t, err)
	require.NotEmpty(t, violations)
	assert.Contains(t, violations[0].Message, "translation_of")
}

func TestValidateCLI_WhenValidTranslations_ExpectOK(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	base := t.TempDir()
	themeDir := filepath.Join(base, "theme")
	_ = os.MkdirAll(themeDir, 0o755)

	originalContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
translations: [article.ru]
---
# Original

[[article.ru|Русский перевод]]
`
	translationContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
translation_of: article
lang: ru
---
# Оригинал

[[article|Original]]
`
	_ = os.WriteFile(filepath.Join(themeDir, "article.md"), []byte(originalContent), 0o644)
	_ = os.WriteFile(filepath.Join(themeDir, "article.ru.md"), []byte(translationContent), 0o644)

	violations, err := kb.Validate(ctx, base)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestValidateCLI_WhenOriginalWithoutTranslations_ExpectViolation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	base := t.TempDir()
	themeDir := filepath.Join(base, "theme")
	_ = os.MkdirAll(themeDir, 0o755)

	originalContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
---
# Original
`
	translationContent := `---
keywords: [go]
created: "2024-01-01"
updated: "2024-01-01"
translation_of: article
lang: ru
---
# Оригинал
`
	_ = os.WriteFile(filepath.Join(themeDir, "article.md"), []byte(originalContent), 0o644)
	_ = os.WriteFile(filepath.Join(themeDir, "article.ru.md"), []byte(translationContent), 0o644)

	violations, err := kb.Validate(ctx, base)

	require.NoError(t, err)
	require.NotEmpty(t, violations)
	assert.Contains(t, violations[0].Message, "translations")
}
