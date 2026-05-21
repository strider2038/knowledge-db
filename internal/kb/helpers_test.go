package kb_test

import (
	"crypto/sha256"
	"path/filepath"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/spf13/afero"
	"github.com/strider2038/knowledge-db/internal/kb"
)

const testValidNodeID = "018f0000-0000-7000-8000-000000000099"

func withTestID(matter map[string]any) map[string]any {
	if matter == nil {
		matter = map[string]any{}
	}
	if _, ok := matter["id"]; !ok {
		matter["id"] = testValidNodeID
	}

	return matter
}

func testNodeIDForPath(path string) string {
	sum := sha256.Sum256([]byte("kb-test-node-id:" + path))
	var b [16]byte
	copy(b[:], sum[:16])
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80

	return strings.ToLower(uuid.Must(uuid.FromBytes(b[:])).String())
}

func testFrontmatterPrefix(body, relPath string) string {
	if strings.Contains(body, "\nid:") || strings.Contains(body, "id:") {
		return body
	}
	id := testNodeIDForPath(relPath)

	return strings.Replace(body, "---\n", "---\nid: \""+id+"\"\n", 1)
}

// seedMemFS создаёт in-memory fs с указанными файлами и возвращает Store и basePath "/".
func seedMemFS(files map[string]string) (*kb.Store, string) {
	fs := afero.NewMemMapFs()
	basePath := "/"
	for path, content := range files {
		fullPath := filepath.Join(basePath, path)
		dir := filepath.Dir(fullPath)
		_ = fs.MkdirAll(dir, 0o755)
		_ = afero.WriteFile(fs, fullPath, []byte(testFrontmatterPrefix(content, path)), 0o644)
	}

	return kb.NewStore(fs), basePath
}
