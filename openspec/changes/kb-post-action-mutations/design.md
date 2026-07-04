# Design: POST-action мутации

## Контекст

knowledge-db осознанно использует REST-адресацию (skill `api-conventions` подтверждён при
Phase-2 flat-vendor split). Данное изменение — **гибрид**, а не полный переход на house-стиль:
мутации приводятся к POST-action, а `GET`-чтения сохраняют адресацию по пути (shareable deep-links
базы знаний — единственное место, где это реально ценно). Решение принято владельцем как
«mutations-only pragmatic».

## Полное отображение маршрутов

| Было | Стало | id/путь → тело |
|------|-------|----------------|
| `PATCH /api/nodes/{path...}` | `POST /api/nodes/update` | `path` + поля patch |
| `DELETE /api/nodes/{path...}` | `POST /api/nodes/delete` | `path` |
| `POST /api/nodes/{path...}/move` | `POST /api/nodes/move` | `path`, `target_path` |
| `POST /api/nodes/{path...}/refresh-description` | `POST /api/nodes/refresh-description` | `path` |
| `POST /api/nodes/{path...}/normalize` | `POST /api/nodes/normalize` | `path` |
| `POST /api/nodes/{path...}/agent-edit` | `POST /api/nodes/agent-edit` | `path` |
| `POST /api/nodes/{path...}/dump-images` | `POST /api/nodes/dump-images` | `path` |
| `PATCH /api/chats/{id}` | `POST /api/chats/update` | `id`, `title` |
| `DELETE /api/chats/{id}` | `POST /api/chats/delete` | `id` |
| `PATCH /api/debug/issues/{id}` | `POST /api/debug/issues/update` | `id` + поля |
| `POST /api/import/telegram/session/{id}/accept` | `POST /api/import/telegram/session/accept` | `id` + поля |
| `POST /api/import/telegram/session/{id}/reject` | `POST /api/import/telegram/session/reject` | `id` |
| `POST /api/articles/translate/{path...}` | `POST /api/articles/translate` | `path` + поля |

**Не меняются (GET-чтения / deep-links):** `GET /api/nodes/{path...}`, `GET /api/nodes/by-id/{id}`,
`GET /api/assets/{path...}`, `GET /api/chats/{id}`, `GET /api/import/telegram/session/{id}`,
`GET /api/articles/translate/{path...}`, все `GET /api/jobs/{id}`, `/logs`,
`GET /api/node-normalization/{id}(/logs)`, `GET /api/node-agent-edit/{id}(/logs)`,
`GET /api/node-dump-images/{id}(/logs)`, `GET /api/tree`, `GET /api/search`, `GET /api/nodes`,
`GET /api/git/status`, `GET /api/index/status`, health, auth OAuth, SPA.

## Ключевые решения

### 1. Удаление suffix-dispatch мультиплексора `MoveNode`

Сейчас `POST /api/nodes/{path...}` — единый обработчик `MoveNode`, который по суффиксу `path`
разветвляется на `refresh-description`/`normalize`/`agent-edit`/`dump-images`, а иначе выполняет
move (обрезая `/move`). После изменения каждое действие регистрируется отдельным маршрутом
`POST /api/nodes/<action>` и получает собственный обработчик. `MoveNode`-мультиплексор и вся
`strings.HasSuffix`/`CutSuffix(path, "/...")`-логика удаляются. Обработчики
`node_normalization.go`, `node_agent_edit.go`, `node_dump_images.go`, `RefreshDescription`
читают `path` из декодированного тела вместо `TrimSuffix(r.PathValue("path"), "/...")`.

### 2. Извлечение `path`/`id` из тела

Общий приём: декодировать тело в структуру с полем `path` (или `id`) + прикладные поля;
`strings.TrimSpace`, при пустом — `400 "path required"` / `"id required"` (сохранить прежние
сообщения). Валидация обхода путей (`..`), `kb.ValidateNodeID` и маппинг доменных ошибок —
**без изменений**.

### 3. `POST /api/nodes/update` (бывший PATCH)

`PatchNode` декодирует тело как `map[string]json.RawMessage` для отличия «поле отсутствует» от
«поле = null/zero». Сохраняем этот приём: сначала извлекаем и удаляем ключ `path` из карты,
затем прежняя валидация допустимых ключей (`manual_processed`, `title`, `keywords`, `labels`)
и разбор — по остатку. Если после удаления `path` карта пуста — `400 "body must contain at least
one field"` (как раньше). Проверка `r.Method != PATCH` убирается (теперь POST-only через маршрут).

### 4. Guard: verbs-only

`internal/api/router_guard_test.go` сканирует `router.go` регуляркой
`(?:HandleFunc|Handle)\("(GET|POST|PUT|DELETE|PATCH) ([^"]+)"` и падает **только** на
`PUT`/`DELETE`/`PATCH`. Проверку `{` под `/api/` **не** включаем (в отличие от agentmem/comm-relay) —
GET deep-links и path-адресация чтений здесь легитимны. Комментарий в тесте фиксирует это
осознанное отличие и ссылается на `.agents/skills/api-conventions/SKILL.md`.

### 5. Frontend `web/src/services/api.ts`

13 вызовов переводятся на новые пути и метод POST, id/путь — в JSON-тело (`method: 'POST'`,
`body: JSON.stringify({ path/id, ... })`). Прежние `encodeURIComponent(path)` в URL уходят
(путь теперь в теле, кодировать не нужно). GET-читатели (`getNode`, `getChat`, translate GET,
telegram session GET, статус-поллеры) не трогаем.

## Non-Goals

- **Не** переводим `GET`-чтения на POST (это был бы «full strict», явно отклонён).
- **Не** выносим `GET /api/assets/{path...}` из `/api/` — download-исключение остаётся.
- **Не** меняем доменный слой `internal/kb/*`, форматы полей (кроме добавления `path`/`id`),
  валидацию, индексацию, job-подсистему.
- **Не** трогаем hub-скилл `api-conventions` — knowledge-db держит свой локальный (гибрид).

## Слайсы (для делегирования)

1. **nodes** (7 маршрутов) — `router.go`, `handlers.go` (update/delete/move + удаление
   мультиплексора), `node_normalization.go`/`node_agent_edit.go`/`node_dump_images.go` (path из тела),
   тесты, `api.ts` node-вызовы.
2. **chats + debug/issues** (3) — `chat_sessions.go`, `debug_issues.go`, `router.go`, тесты, `api.ts`.
3. **import/telegram + articles/translate** (3) — `import_handlers.go`, `translate_handlers.go`,
   `router.go`, тесты, `api.ts`.
4. **guard + docs** — `router_guard_test.go` (verbs-only), `AGENTS.md`, локальный скилл
   `api-conventions`. Guard добавляется **последним**, когда все мутации уже POST-action.
