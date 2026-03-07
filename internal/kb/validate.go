package kb

import (
	"context"

	"github.com/spf13/afero"
)

const maxDepth = 3

// Validate проверяет структуру базы: темы 2–3 уровня, узлы как .md файлы с frontmatter.
func Validate(ctx context.Context, basePath string) ([]ValidationError, error) {
	return NewStore(afero.NewOsFs()).Validate(ctx, basePath)
}

// IsNode проверяет, является ли стем-путь узлом (существует {stem}.md).
func IsNode(stemPath string) bool {
	return NewStore(afero.NewOsFs()).isNode(stemPath)
}

// MaxThemeDepth возвращает максимально допустимую глубину тем.
func MaxThemeDepth() int {
	return maxDepth
}
