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

// Validate проверяет структуру базы: темы 2–3 уровня, узлы как .md файлы с frontmatter.
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
			// Если рядом существует {dirname}.md — это директория вложений, пропускаем.
			if _, statErr := s.fs.Stat(path + ".md"); statErr == nil {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				violations = append(violations, ValidationError{Path: rel, Message: "theme depth exceeds 2-3 levels"})

				return filepath.SkipDir
			}

			return nil
		}

		// Файл: проверяем только .md узлы.
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		stemRel := strings.TrimSuffix(rel, ".md")
		if err := s.validateNode(path, stemRel, &violations); err != nil {
			return err
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
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		slug := strings.TrimSuffix(name, ".md")
		nodeRel := filepath.ToSlash(filepath.Join(themePath, slug))
		nodes = append(nodes, &TreeNode{
			Name: slug,
			Path: nodeRel,
		})
	}

	return nodes, nil
}

// IsNode проверяет, является ли стем-путь узлом (существует {stem}.md).
func (s *Store) IsNode(stemPath string) bool {
	return s.isNode(stemPath)
}

// ListAllNodes рекурсивно возвращает все узлы базы знаний.
func (s *Store) ListAllNodes(ctx context.Context, basePath string) ([]*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	var nodes []*TreeNode
	err := afero.Walk(s.fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(basePath, path)
		slug := strings.TrimSuffix(info.Name(), ".md")
		stemRel := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		nodes = append(nodes, &TreeNode{
			Name: slug,
			Path: stemRel,
		})

		return nil
	})
	if err != nil {
		return nil, errors.Errorf("list all nodes: %w", err)
	}

	return nodes, nil
}

// GetNode читает узел по пути стема (relative path от корня базы, без расширения .md).
func (s *Store) GetNode(ctx context.Context, basePath, nodePath string) (*Node, error) {
	basePath = filepath.Clean(basePath)
	stemPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !s.isNode(stemPath) {
		return nil, errors.Errorf("get node: %w", ErrNodeNotFound)
	}

	meta, annotation, content, err := parseNodeFile(s.fs, stemPath)
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

// isNode проверяет, существует ли {stemPath}.md.
func (s *Store) isNode(stemPath string) bool {
	_, err := s.fs.Stat(stemPath + ".md")

	return err == nil
}

func (s *Store) validateNode(filePath, stemRel string, violations *[]ValidationError) error {
	data, err := afero.ReadFile(s.fs, filePath)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "cannot read file: " + err.Error()})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	matter, err := parseFrontmatter(data)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "invalid frontmatter: " + err.Error()})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	if msg := ValidateFrontmatter(matter); msg != "" {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: msg})
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
		// Пропускаем директории вложений: если рядом существует {name}.md.
		if _, err := s.fs.Stat(filepath.Join(currentPath, name+".md")); err == nil {
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
