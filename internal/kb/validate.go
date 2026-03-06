package kb

import (
	"context"

	"github.com/spf13/afero"
)

const maxDepth = 3

// Validate проверяет структуру базы: темы 2–3 уровня, узлы с {dirname}.md и frontmatter.
func Validate(ctx context.Context, basePath string) ([]ValidationError, error) {
	return NewStore(afero.NewOsFs()).Validate(ctx, basePath)
}

// IsNodeDir проверяет, является ли директория узлом (содержит {dirname}.md с frontmatter).
func IsNodeDir(path string) bool {
	return NewStore(afero.NewOsFs()).isNode(path)
}

// MaxThemeDepth возвращает максимально допустимую глубину тем.
func MaxThemeDepth() int {
	return maxDepth
}
