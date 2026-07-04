# Proposal: POST-action мутации в HTTP API

## Why

HTTP API knowledge-db смешивает стили: чтения адресуются по пути (`GET /api/nodes/{path...}`),
а мутации используют REST-глаголы (`PATCH`/`DELETE`) и суффиксную диспетчеризацию
(`POST /api/nodes/{path...}/move|normalize|...` через один wildcard-обработчик `MoveNode`).
Это расходится с «домашним» стилем остальных веб-проектов (POST-action / RPC) и усложняет
единый route-guard. Приводим **мутации** к единому виду `POST /api/<resource>/<action>` с
идентификатором/путём в JSON-теле, сохраняя REST-адресацию **только для чтений** (shareable
deep-links базы знаний — единственное место, где адресация по пути реально полезна).

## What Changes

- **BREAKING (внутренний API + web-клиент):** мутации переходят на `POST /api/<resource>/<action>`:
  - `PATCH /api/nodes/{path...}` → `POST /api/nodes/update`
  - `DELETE /api/nodes/{path...}` → `POST /api/nodes/delete`
  - `POST /api/nodes/{path...}/move` → `POST /api/nodes/move`
  - `POST /api/nodes/{path...}/refresh-description` → `POST /api/nodes/refresh-description`
  - `POST /api/nodes/{path...}/normalize` → `POST /api/nodes/normalize`
  - `POST /api/nodes/{path...}/agent-edit` → `POST /api/nodes/agent-edit`
  - `POST /api/nodes/{path...}/dump-images` → `POST /api/nodes/dump-images`
  - `PATCH /api/chats/{id}` → `POST /api/chats/update`
  - `DELETE /api/chats/{id}` → `POST /api/chats/delete`
  - `PATCH /api/debug/issues/{id}` → `POST /api/debug/issues/update`
  - `POST /api/import/telegram/session/{id}/accept` → `POST /api/import/telegram/session/accept`
  - `POST /api/import/telegram/session/{id}/reject` → `POST /api/import/telegram/session/reject`
  - `POST /api/articles/translate/{path...}` → `POST /api/articles/translate`
  - Идентификатор/путь переезжает в тело: `path`, `id`, `target_path` (snake_case).
- Удаляется suffix-dispatch мультиплексор `MoveNode` — каждое действие получает свой маршрут и
  обработчик; статус-обработчики читают `path` из тела вместо `TrimSuffix(PathValue, "/...")`.
- **Чтения не меняются.** `GET /api/nodes/{path...}`, `GET /api/nodes/by-id/{id}`,
  `GET /api/assets/{path...}`, `GET /api/chats/{id}`, все `GET`-поллеры статуса/логов
  (`node-normalization`, `node-agent-edit`, `node-dump-images`, `jobs`), `GET /api/import/telegram/session/{id}`,
  `GET /api/articles/translate/{path...}` — остаются как есть (санкционированное REST-исключение
  для shareable deep-links).
- Route-guard тест смягчается до **verbs-only**: падает на `PUT`/`DELETE`/`PATCH` под `/api/`,
  но допускает `{...}` в `GET`-маршрутах.
- Обновляются `AGENTS.md` и локальный скилл `api-conventions`: фиксируем гибрид —
  «мутации POST-action, чтения REST-по-пути».

## Capabilities

- **Modified Capabilities:**
  - `rest-api` — меняется контракт маршрутов мутаций (метод, путь, расположение идентификатора).

## Impact

- **Backend:** `internal/api/router.go`, `handlers.go`, `chat_sessions.go`, `debug_issues.go`,
  `import_handlers.go`, `translate_handlers.go`, `node_normalization.go`, `node_agent_edit.go`,
  `node_dump_images.go`; новый `router_guard_test.go`; обновление `*_test.go`.
- **Frontend:** `web/src/services/api.ts` (13 вызовов).
- **Docs/skills:** `AGENTS.md`, `.agents/skills/api-conventions/SKILL.md`.
- **Без изменений:** доменный слой `internal/kb/*`, форматы тел (кроме добавления `path`/`id`),
  логика валидации/обхода путей, все `GET`-чтения и статус-поллеры.
