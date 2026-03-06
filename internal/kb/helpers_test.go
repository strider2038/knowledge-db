package kb_test

import (
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// seedMemFS создаёт in-memory fs с указанными файлами и возвращает Store и basePath "/".
// paths — относительные пути от корня (например "topic/node1/node1.md").
func seedMemFS(files map[string]string) (*kb.Store, string) {
	fs := afero.NewMemMapFs()
	basePath := "/"
	for path, content := range files {
		fullPath := filepath.Join(basePath, path)
		dir := filepath.Dir(fullPath)
		_ = fs.MkdirAll(dir, 0o755)
		_ = afero.WriteFile(fs, fullPath, []byte(content), 0o644)
	}

	return kb.NewStore(fs), basePath
}
