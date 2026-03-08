package kb

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

// CreateTranslationFile создаёт файл перевода {themePath}/{slug}.{lang}.md.
func (s *Store) CreateTranslationFile(
	ctx context.Context,
	basePath, themePath, slug, lang string,
	frontmatter map[string]any,
	content string,
) error {
	basePath = filepath.Clean(basePath)
	themeDir := filepath.Join(basePath, filepath.FromSlash(themePath))
	if err := s.fs.MkdirAll(themeDir, 0o755); err != nil {
		return errors.Errorf("create translation file: mkdir: %w", err)
	}

	fmBytes, err := FormatFrontmatter(frontmatter)
	if err != nil {
		return errors.Errorf("create translation file: %w", err)
	}

	var fileContent []byte
	fileContent = append(fileContent, fmBytes...)
	if content != "" {
		fileContent = append(fileContent, '\n')
		fileContent = append(fileContent, []byte(content)...)
		fileContent = append(fileContent, '\n')
	}

	fileName := fmt.Sprintf("%s.%s.md", slug, lang)
	mdPath := filepath.Join(themeDir, fileName)
	if err := afero.WriteFile(s.fs, mdPath, fileContent, 0o644); err != nil {
		return errors.Errorf("create translation file: write: %w", err)
	}

	return nil
}

// AppendTranslationsToOriginal читает оригинал, добавляет translations в frontmatter
// и wikilink в тело, перезаписывает файл.
func (s *Store) AppendTranslationsToOriginal(
	ctx context.Context,
	basePath, themePath, slug, translationSlug string,
) error {
	basePath = filepath.Clean(basePath)
	stemPath := filepath.Join(basePath, filepath.FromSlash(themePath), slug)

	matter, _, content, err := parseNodeFile(s.fs, stemPath)
	if err != nil {
		return errors.Errorf("append translations: read original: %w", err)
	}

	// Добавляем или обновляем translations в frontmatter.
	var translations []string
	if existing, ok := matter["translations"]; ok {
		switch v := existing.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					translations = append(translations, s)
				}
			}
		case []string:
			translations = v
		}
	}
	// Проверяем, что translationSlug ещё не в списке.
	found := slices.Contains(translations, translationSlug)
	if !found {
		translations = append(translations, translationSlug)
	}
	matter["translations"] = translations

	// Добавляем wikilink в конец тела, если его ещё нет.
	wikilink := fmt.Sprintf("[[%s|Русский перевод]]", translationSlug)
	if !strings.Contains(content, wikilink) {
		content = strings.TrimSuffix(content, "\n") + "\n\n" + wikilink + "\n"
	}

	fmBytes, err := FormatFrontmatter(matter)
	if err != nil {
		return errors.Errorf("append translations: format frontmatter: %w", err)
	}

	var fileContent []byte
	fileContent = append(fileContent, fmBytes...)
	if content != "" {
		fileContent = append(fileContent, '\n')
		fileContent = append(fileContent, []byte(content)...)
		fileContent = append(fileContent, '\n')
	}

	mdPath := stemPath + ".md"
	if err := afero.WriteFile(s.fs, mdPath, fileContent, 0o644); err != nil {
		return errors.Errorf("append translations: write: %w", err)
	}

	return nil
}
