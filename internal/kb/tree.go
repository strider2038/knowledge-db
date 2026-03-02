package kb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/muonsoft/errors"
)

// ReadTree возвращает дерево тем и подтем базы знаний.
func ReadTree(ctx context.Context, basePath string) (*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	info, err := os.Stat(basePath)
	if err != nil {
		return nil, errors.Errorf("read tree: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("read tree: %w", ErrInvalidPath)
	}

	root := &TreeNode{Name: "", Path: ""}
	if err := buildTree(basePath, basePath, "", root, 0); err != nil {
		return nil, err
	}
	return root, nil
}

func buildTree(basePath, currentPath, relPath string, parent *TreeNode, depth int) error {
	if depth > maxDepth {
		return nil
	}
	entries, err := os.ReadDir(currentPath)
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
		if IsNodeDir(childPath) {
			continue
		}
		child := &TreeNode{
			Name: name,
			Path: filepath.ToSlash(childRel),
		}
		if err := buildTree(basePath, childPath, childRel, child, depth+1); err != nil {
			return err
		}
		parent.Children = append(parent.Children, child)
	}
	return nil
}

// ListNodes возвращает список узлов по пути темы (path — относительный путь от корня базы).
func ListNodes(ctx context.Context, basePath, themePath string) ([]*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	fullPath := filepath.Join(basePath, filepath.FromSlash(themePath))
	info, err := os.Stat(fullPath)
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
	entries, err := os.ReadDir(fullPath)
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
		if IsNodeDir(childPath) {
			nodeRel := filepath.Join(themePath, name)
			nodes = append(nodes, &TreeNode{
				Name: name,
				Path: filepath.ToSlash(nodeRel),
			})
		}
	}
	return nodes, nil
}

// GetNode читает узел по пути (relative path от корня базы).
func GetNode(ctx context.Context, basePath, nodePath string) (*Node, error) {
	basePath = filepath.Clean(basePath)
	fullPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !IsNodeDir(fullPath) {
		return nil, errors.Errorf("get node: %w", ErrNodeNotFound)
	}

	annotation, _ := os.ReadFile(filepath.Join(fullPath, "annotation.md"))
	content, _ := os.ReadFile(filepath.Join(fullPath, "content.md"))
	metaData, err := os.ReadFile(filepath.Join(fullPath, "metadata.json"))
	if err != nil {
		return nil, errors.Errorf("get node: %w", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, errors.Errorf("get node: %w", err)
	}
	return &Node{
		Path:       filepath.ToSlash(nodePath),
		Annotation: string(annotation),
		Content:    string(content),
		Metadata:   meta,
	}, nil
}
