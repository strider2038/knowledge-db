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

// PatchNodeManualProcessed обновляет флаг manual_processed в файле узла.
func PatchNodeManualProcessed(ctx context.Context, basePath, nodePath string, value bool) error {
	return NewStore(afero.NewOsFs()).PatchNodeManualProcessed(ctx, basePath, nodePath, value)
}

// DeleteNode удаляет узел из базы знаний (файл .md и вложения).
func DeleteNode(ctx context.Context, basePath, nodePath string) error {
	return NewStore(afero.NewOsFs()).DeleteNode(ctx, basePath, nodePath)
}

// MoveNode перемещает узел по указанному целевому пути.
func MoveNode(ctx context.Context, basePath, nodePath, targetPath string) (*Node, error) {
	return NewStore(afero.NewOsFs()).MoveNode(ctx, basePath, nodePath, targetPath)
}
