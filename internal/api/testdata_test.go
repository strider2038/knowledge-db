package api_test

import (
	"strings"

	"github.com/strider2038/knowledge-db/internal/index/sqlite"
)

func injectTestNodeID(yaml, path string) string {
	if strings.Contains(yaml, "\nid:") {
		return yaml
	}
	id := sqlite.TestNodeID(strings.TrimSuffix(path, ".md"))

	return strings.Replace(yaml, "---\n", "---\nid: \""+id+"\"\n", 1)
}
