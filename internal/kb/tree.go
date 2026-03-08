package kb

import (
	"context"

	"github.com/spf13/afero"
)

// ReadTree возвращает дерево тем и подтем базы знаний.
func ReadTree(ctx context.Context, basePath string) (*TreeNode, error) {
	return NewStore(afero.NewOsFs()).ReadTree(ctx, basePath)
}

// ListNodes возвращает список узлов по пути темы (path — относительный путь от корня базы).
func ListNodes(ctx context.Context, basePath, themePath string) ([]*TreeNode, error) {
	return NewStore(afero.NewOsFs()).ListNodes(ctx, basePath, themePath)
}

// ListNodesWithOptions возвращает список узлов с фильтрами, поиском, сортировкой и пагинацией.
func ListNodesWithOptions(ctx context.Context, basePath string, opts ListNodesOptions) ([]*NodeListItem, int, error) {
	return NewStore(afero.NewOsFs()).ListNodesWithOptions(ctx, basePath, opts)
}

// GetNode читает узел по пути (relative path от корня базы).
func GetNode(ctx context.Context, basePath, nodePath string) (*Node, error) {
	return NewStore(afero.NewOsFs()).GetNode(ctx, basePath, nodePath)
}
