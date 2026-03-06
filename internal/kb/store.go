package kb

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

// Store — хранилище базы знаний с абстракцией файловой системы.
// Позволяет использовать in-memory fs в тестах (afero.MemMapFs).
type Store struct {
	fs afero.Fs
}

// NewStore создаёт Store с указанной файловой системой.
// Для production: afero.NewOsFs()
// Для тестов: afero.NewMemMapFs().
func NewStore(fs afero.Fs) *Store {
	return &Store{fs: fs}
}

// Validate проверяет структуру базы: темы 2–3 уровня, узлы с {dirname}.md и frontmatter.
func (s *Store) Validate(ctx context.Context, basePath string) ([]ValidationError, error) {
	basePath = filepath.Clean(basePath)
	info, err := s.fs.Stat(basePath)
	if err != nil {
		return nil, errors.Errorf("validate base: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("validate base: %w", ErrInvalidPath)
	}

	var violations []ValidationError
	err = afero.Walk(s.fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(basePath, path)
		if rel == "." {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		depth := len(parts)
		if info.IsDir() {
			if depth > maxDepth {
				violations = append(violations, ValidationError{Path: rel, Message: "theme depth exceeds 2-3 levels"})

				return filepath.SkipDir
			}
			if s.isNode(path) {
				if err := s.validateNode(path, rel, &violations); err != nil {
					return err
				}

				return filepath.SkipDir
			}

			return nil
		}

		return nil
	})
	if err != nil {
		return nil, errors.Errorf("validate base: %w", err)
	}

	return violations, nil
}

// ReadTree возвращает дерево тем и подтем базы знаний.
func (s *Store) ReadTree(ctx context.Context, basePath string) (*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	info, err := s.fs.Stat(basePath)
	if err != nil {
		return nil, errors.Errorf("read tree: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("read tree: %w", ErrInvalidPath)
	}

	root := &TreeNode{Name: "", Path: ""}
	if err := s.buildTree(basePath, basePath, "", root, 0); err != nil {
		return nil, err
	}

	return root, nil
}

// ListNodes возвращает список узлов по пути темы.
func (s *Store) ListNodes(ctx context.Context, basePath, themePath string) ([]*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	fullPath := filepath.Join(basePath, filepath.FromSlash(themePath))
	info, err := s.fs.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Errorf("list nodes: %w", ErrNodeNotFound)
		}

		return nil, errors.Errorf("list nodes: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("list nodes: %w", ErrInvalidPath)
	}

	var nodes []*TreeNode
	entries, err := s.readDir(fullPath)
	if err != nil {
		return nil, errors.Errorf("list nodes: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		childPath := filepath.Join(fullPath, name)
		if s.isNode(childPath) {
			nodeRel := filepath.Join(themePath, name)
			nodes = append(nodes, &TreeNode{
				Name: name,
				Path: filepath.ToSlash(nodeRel),
			})
		}
	}

	return nodes, nil
}

// IsNodeDir проверяет, является ли директория узлом (содержит {dirname}.md).
func (s *Store) IsNodeDir(path string) bool {
	return s.isNode(path)
}

// GetNode читает узел по пути (relative path от корня базы).
func (s *Store) GetNode(ctx context.Context, basePath, nodePath string) (*Node, error) {
	basePath = filepath.Clean(basePath)
	fullPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !s.isNode(fullPath) {
		return nil, errors.Errorf("get node: %w", ErrNodeNotFound)
	}

	meta, annotation, content, err := parseNodeFile(s.fs, fullPath)
	if err != nil {
		return nil, errors.Errorf("get node: %w", err)
	}

	return &Node{
		Path:       filepath.ToSlash(nodePath),
		Annotation: annotation,
		Content:    content,
		Metadata:   meta,
	}, nil
}

func (s *Store) isNode(path string) bool {
	dirname := filepath.Base(path)
	mainPath := filepath.Join(path, dirname+".md")
	_, err := s.fs.Stat(mainPath)

	return err == nil
}

func (s *Store) validateNode(nodePath, rel string, violations *[]ValidationError) error {
	dirname := filepath.Base(nodePath)
	mainPath := filepath.Join(nodePath, dirname+".md")
	if _, err := s.fs.Stat(mainPath); err != nil {
		*violations = append(*violations, ValidationError{Path: rel, Message: "missing " + dirname + ".md"})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	data, err := afero.ReadFile(s.fs, mainPath)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: rel, Message: "cannot read " + dirname + ".md"})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	matter, err := parseFrontmatter(data)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: rel, Message: "invalid frontmatter: " + err.Error()})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	if msg := ValidateFrontmatter(matter); msg != "" {
		*violations = append(*violations, ValidationError{Path: rel, Message: msg})
	}

	return nil
}

func (s *Store) readDir(path string) ([]os.FileInfo, error) {
	f, err := s.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdir(-1)
}

func (s *Store) buildTree(basePath, currentPath, relPath string, parent *TreeNode, depth int) error { //nolint:unparam // basePath passed to recursive calls
	if depth > maxDepth {
		return nil
	}
	entries, err := s.readDir(currentPath)
	if err != nil {
		return errors.Errorf("read dir %s: %w", currentPath, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		childPath := filepath.Join(currentPath, name)
		childRel := filepath.Join(relPath, name)
		if s.isNode(childPath) {
			continue
		}
		child := &TreeNode{
			Name: name,
			Path: filepath.ToSlash(childRel),
		}
		if err := s.buildTree(basePath, childPath, childRel, child, depth+1); err != nil {
			return err
		}
		parent.Children = append(parent.Children, child)
	}

	return nil
}
