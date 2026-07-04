package api

import (
	"os"
	"regexp"
	"testing"
)

// TestRoutes_PostActionMutations фиксирует «домашний» POST-action контракт мутаций:
// ни один маршрут под /api/ не должен использовать REST-глаголы PUT/DELETE/PATCH.
//
// В ОТЛИЧИЕ от agentmem/comm-relay здесь НЕ запрещается адресация по пути ({id}, {path...}):
// knowledge-db — санкционированное гибридное исключение, где GET-чтения остаются REST по пути
// (shareable deep-links базы знаний, скачивание ассетов, поллинг статуса/логов). Мутации же
// приведены к POST /api/<resource>/<action> с идентификатором/путём в JSON-теле.
// См. .agents/skills/api-conventions/SKILL.md.
func TestRoutes_PostActionMutations(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}

	re := regexp.MustCompile(`(?:HandleFunc|Handle)\("(GET|POST|PUT|DELETE|PATCH) ([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("no routes matched in router.go — regex may be broken")
	}

	restVerbs := map[string]bool{"PUT": true, "DELETE": true, "PATCH": true}
	for _, m := range matches {
		method, path := m[1], m[2]
		if restVerbs[method] {
			t.Errorf(
				"route %s %s uses REST verb %s; мутации knowledge-db используют POST-action "+
					"(POST /api/<resource>/<action> с id/путём в теле) — см. .agents/skills/api-conventions/SKILL.md",
				method, path, method,
			)
		}
	}
}
