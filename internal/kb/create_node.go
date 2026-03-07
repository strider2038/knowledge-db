package kb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

// CreateNodeParams — параметры для создания нового узла.
type CreateNodeParams struct {
	ThemePath   string
	Slug        string
	Frontmatter map[string]any
	Content     string
}

// CreateNode создаёт новый узел в базе знаний.
// Создаёт файл {basePath}/{themePath}/{slug}.md (без slug-директории).
// При slug-коллизии добавляет суффикс -2, -3 и т.д.
// Возвращает созданный Node.
func (s *Store) CreateNode(ctx context.Context, basePath string, params CreateNodeParams) (*Node, error) {
	basePath = filepath.Clean(basePath)

	slug, err := s.resolveSlug(basePath, params.ThemePath, params.Slug)
	if err != nil {
		return nil, errors.Errorf("create node: %w", err)
	}

	themeDir := filepath.Join(basePath, filepath.FromSlash(params.ThemePath))
	if err := s.fs.MkdirAll(themeDir, 0o755); err != nil {
		return nil, errors.Errorf("create node: mkdir: %w", err)
	}

	fmBytes, err := FormatFrontmatter(params.Frontmatter)
	if err != nil {
		return nil, errors.Errorf("create node: %w", err)
	}

	var fileContent []byte
	fileContent = append(fileContent, fmBytes...)
	if params.Content != "" {
		fileContent = append(fileContent, '\n')
		fileContent = append(fileContent, []byte(params.Content)...)
		fileContent = append(fileContent, '\n')
	}

	mdPath := filepath.Join(themeDir, slug+".md")
	if err := afero.WriteFile(s.fs, mdPath, fileContent, 0o644); err != nil {
		return nil, errors.Errorf("create node: write file: %w", err)
	}

	nodePath := filepath.ToSlash(filepath.Join(params.ThemePath, slug))
	annotation, _ := params.Frontmatter["annotation"].(string)

	return &Node{
		Path:       nodePath,
		Annotation: annotation,
		Content:    params.Content,
		Metadata:   params.Frontmatter,
	}, nil
}

// resolveSlug возвращает slug без коллизии.
// Если {basePath}/{themePath}/{slug}.md уже существует — пробует {slug}-2, {slug}-3 и т.д.
func (s *Store) resolveSlug(basePath, themePath, slug string) (string, error) {
	themeDir := filepath.Join(basePath, filepath.FromSlash(themePath))
	candidate := slug
	for i := 2; i <= 100; i++ {
		mdPath := filepath.Join(themeDir, candidate+".md")
		if _, err := s.fs.Stat(mdPath); err != nil {
			return candidate, nil //nolint:nilerr // err here means file does not exist — no collision
		}
		candidate = fmt.Sprintf("%s-%d", slug, i)
	}

	return "", errors.Errorf("cannot resolve slug %q: too many collisions", slug)
}
