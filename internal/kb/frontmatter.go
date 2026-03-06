package kb

import (
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

// nodeMainFile возвращает путь к главному .md файлу узла (dirname.md).
func nodeMainFile(nodePath string) string {
	dirname := filepath.Base(nodePath)

	return filepath.Join(nodePath, dirname+".md")
}

// parseFrontmatter парсит YAML frontmatter из данных файла.
func parseFrontmatter(data []byte) (map[string]any, error) {
	var matter map[string]any
	if _, err := frontmatter.Parse(strings.NewReader(string(data)), &matter); err != nil {
		return nil, err
	}

	return matter, nil
}

// parseNodeFile парсит главный .md файл узла через указанную fs.
func parseNodeFile(fs afero.Fs, nodePath string) (map[string]any, string, string, error) {
	mainPath := nodeMainFile(nodePath)
	data, err := afero.ReadFile(fs, mainPath)
	if err != nil {
		return nil, "", "", errors.Errorf("read node file: %w", err)
	}
	var matter map[string]any
	rest, err := frontmatter.Parse(strings.NewReader(string(data)), &matter)
	if err != nil {
		return nil, "", "", errors.Errorf("parse frontmatter: %w", err)
	}
	annotation, _ := matter["annotation"].(string)
	content := strings.TrimSpace(string(rest))

	return matter, annotation, content, nil
}

// ParseNodeFile парсит главный .md файл узла: frontmatter → metadata + annotation, тело → content.
// Использует реальную ФС (обёртка для обратной совместимости).
func ParseNodeFile(nodePath string) (map[string]any, string, string, error) {
	return parseNodeFile(afero.NewOsFs(), nodePath)
}

// ValidateFrontmatter проверяет наличие обязательных полей (keywords, created, updated).
// Возвращает nil при успехе, иначе описание ошибки.
func ValidateFrontmatter(matter map[string]any) string {
	if matter == nil {
		return "frontmatter required"
	}
	if _, ok := matter["keywords"]; !ok {
		return "frontmatter: keywords required"
	}
	if v, ok := matter["created"]; !ok || v == nil || v == "" {
		return "frontmatter: created required"
	}
	if v, ok := matter["updated"]; !ok || v == nil || v == "" {
		return "frontmatter: updated required"
	}

	return ""
}
