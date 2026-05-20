package kb

import (
	"context"
	"path/filepath"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

type NodeFile struct {
	Path        string
	Frontmatter map[string]any
	Content     string
}

type UpdateNodeParams struct {
	Frontmatter map[string]any
	Content     string
}

func (s *Store) GetNodeFile(ctx context.Context, basePath, nodePath string) (*NodeFile, error) {
	_ = ctx
	basePath = filepath.Clean(basePath)
	stemPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !s.isNode(stemPath) {
		return nil, errors.Errorf("get node file: %w", ErrNodeNotFound)
	}

	matter, _, content, err := parseNodeFile(s.fs, stemPath)
	if err != nil {
		return nil, errors.Errorf("get node file: %w", err)
	}

	return &NodeFile{
		Path:        filepath.ToSlash(nodePath),
		Frontmatter: matter,
		Content:     content,
	}, nil
}

func (s *Store) UpdateNode(ctx context.Context, basePath, nodePath string, params UpdateNodeParams) (*Node, error) {
	_ = ctx
	basePath = filepath.Clean(basePath)
	stemPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !s.isNode(stemPath) {
		return nil, errors.Errorf("update node: %w", ErrNodeNotFound)
	}
	if msg := ValidateFrontmatter(params.Frontmatter); msg != "" {
		return nil, errors.Errorf("update node: invalid frontmatter: %s", msg)
	}

	fmBytes, err := FormatFrontmatter(params.Frontmatter)
	if err != nil {
		return nil, errors.Errorf("update node: %w", err)
	}

	var fileContent []byte
	fileContent = append(fileContent, fmBytes...)
	if params.Content != "" {
		fileContent = append(fileContent, '\n')
		fileContent = append(fileContent, []byte(params.Content)...)
		fileContent = append(fileContent, '\n')
	}

	mdPath := stemPath + ".md"
	tmpPath := mdPath + ".tmp"
	if err := afero.WriteFile(s.fs, tmpPath, fileContent, 0o644); err != nil {
		return nil, errors.Errorf("update node: write: %w", err)
	}
	if err := s.fs.Rename(tmpPath, mdPath); err != nil {
		_ = s.fs.Remove(tmpPath)

		return nil, errors.Errorf("update node: rename: %w", err)
	}

	annotation, _ := params.Frontmatter["annotation"].(string)

	return &Node{
		ID:         NodeIDFromMetadata(params.Frontmatter),
		Path:       filepath.ToSlash(nodePath),
		Annotation: annotation,
		Content:    params.Content,
		Metadata:   NormalizeNodeMetadataForAPI(params.Frontmatter),
	}, nil
}
