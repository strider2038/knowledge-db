package kb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/muonsoft/errors"
)

const maxDepth = 3

// Validate проверяет структуру базы: темы 2–3 уровня, узлы с annotation.md, content.md, metadata.json.
func Validate(ctx context.Context, basePath string) ([]ValidationError, error) {
	basePath = filepath.Clean(basePath)
	info, err := os.Stat(basePath)
	if err != nil {
		return nil, errors.Errorf("validate base: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("validate base: %w", ErrInvalidPath)
	}

	var violations []ValidationError
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
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
			if isNode(path) {
				if err := validateNode(path, rel, &violations); err != nil {
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

func isNode(path string) bool {
	annotation := filepath.Join(path, "annotation.md")
	content := filepath.Join(path, "content.md")
	metadata := filepath.Join(path, "metadata.json")
	_, a := os.Stat(annotation)
	_, c := os.Stat(content)
	_, m := os.Stat(metadata)
	return a == nil && c == nil && m == nil
}

func validateNode(nodePath, rel string, violations *[]ValidationError) error {
	required := []string{"annotation.md", "content.md", "metadata.json"}
	for _, name := range required {
		p := filepath.Join(nodePath, name)
		if _, err := os.Stat(p); err != nil {
			*violations = append(*violations, ValidationError{Path: rel, Message: "missing " + name})
			return nil
		}
	}
	metaPath := filepath.Join(nodePath, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: rel, Message: "cannot read metadata.json"})
		return nil
	}
	var meta struct {
		Keywords []string `json:"keywords"`
		Created  string   `json:"created"`
		Updated  string   `json:"updated"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		*violations = append(*violations, ValidationError{Path: rel, Message: "invalid metadata.json: " + err.Error()})
		return nil
	}
	if meta.Keywords == nil {
		*violations = append(*violations, ValidationError{Path: rel, Message: "metadata.json: keywords required"})
	}
	if meta.Created == "" {
		*violations = append(*violations, ValidationError{Path: rel, Message: "metadata.json: created required"})
	}
	if meta.Updated == "" {
		*violations = append(*violations, ValidationError{Path: rel, Message: "metadata.json: updated required"})
	}
	return nil
}

// IsNodeDir проверяет, является ли директория узлом (содержит annotation.md, content.md, metadata.json).
func IsNodeDir(path string) bool {
	return isNode(path)
}

// MaxThemeDepth возвращает максимально допустимую глубину тем.
func MaxThemeDepth() int {
	return maxDepth
}
