# Tasks

## 1. Слайс: nodes (7 маршрутов)

- [ ] 1.1 `router.go`: заменить `PATCH/DELETE/POST /api/nodes/{path...}` и его суффиксы на
  `POST /api/nodes/update|delete|move|refresh-description|normalize|agent-edit|dump-images`
- [ ] 1.2 `handlers.go`: `update`/`delete`/`move` читают `path` (и `target_path`) из тела;
  удалить suffix-dispatch мультиплексор `MoveNode`
- [ ] 1.3 `node_normalization.go`/`node_agent_edit.go`/`node_dump_images.go` + `RefreshDescription`:
  читать `path` из тела вместо `TrimSuffix(PathValue, "/...")`
- [ ] 1.4 Обновить тесты node-мутаций (метод POST, путь в теле)
- [ ] 1.5 `web/src/services/api.ts`: node-вызовы (update/delete/move/refresh/normalize/agent-edit/dump-images)
- [ ] 1.6 Проверка: `gofmt`, `go build ./...`, `go test ./internal/api/...`, `task web:build`

## 2. Слайс: chats + debug/issues (3 маршрута)

- [ ] 2.1 `router.go`: `POST /api/chats/update|delete`, `POST /api/debug/issues/update`
- [ ] 2.2 `chat_sessions.go`: `PatchChatByID`→update, `DeleteChatByID`→delete (id из тела)
- [ ] 2.3 `debug_issues.go`: `PatchDebugIssueStatus`→update (id из тела)
- [ ] 2.4 Обновить тесты; `api.ts` chat/debug-вызовы
- [ ] 2.5 Проверка сборки/тестов

## 3. Слайс: import/telegram + articles/translate (3 маршрута)

- [ ] 3.1 `router.go`: `POST /api/import/telegram/session/accept|reject`, `POST /api/articles/translate`
- [ ] 3.2 `import_handlers.go`: accept/reject читают `id` из тела
- [ ] 3.3 `translate_handlers.go`: POST-перевод читает `path` из тела (GET-чтение не трогать)
- [ ] 3.4 Обновить тесты; `api.ts` telegram/translate POST-вызовы
- [ ] 3.5 Проверка сборки/тестов

## 4. Слайс: guard + docs

- [ ] 4.1 `internal/api/router_guard_test.go`: verbs-only guard (падает на PUT/DELETE/PATCH,
  разрешает `{` в GET)
- [ ] 4.2 `AGENTS.md`: строка про POST-action мутации + REST GET-чтения
- [ ] 4.3 `.agents/skills/api-conventions/SKILL.md`: зафиксировать гибрид
- [ ] 4.4 Полная проверка: `go test ./...`, `task web:build`; guard проходит
- [ ] 4.5 `openspec archive kb-post-action-mutations`
