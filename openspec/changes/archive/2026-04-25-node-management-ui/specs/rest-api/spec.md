## MODIFIED Requirements

### Requirement: CRUD узлов

API MUST предоставлять эндпоинты для создания, чтения, обновления и удаления узлов (в scaffold — каркас/заглушки). Добавляется поддержка DELETE для удаления узла и POST /move для перемещения.

#### Сценарий: Получение узла по пути

- **WHEN** GET /api/nodes/{path}
- **THEN** возвращается узел или 404

#### Сценарий: Получение дерева тем

- **WHEN** GET /api/tree
- **THEN** возвращается иерархическое дерево тем и подтем

#### Сценарий: Удаление узла

- **WHEN** DELETE /api/nodes/{path}
- **THEN** узел (файл .md и директория вложений) удаляется, возвращается `{ path, deleted: true }` или 404

#### Сценарий: Перемещение узла

- **WHEN** POST /api/nodes/{path}/move с `{ target_path: "new/topic/slug" }`
- **THEN** узел перемещается по указанному пути, промежуточные директории создаются рекурсивно, возвращается обновлённый объект узла, 409 при конфликте

## ADDED Requirements

### Requirement: Git-статус API

API MUST предоставлять эндпоинт `GET /api/git/status`, возвращающий информацию о незакоммиченных изменениях в git-репозитории базы. Ответ MUST содержать `has_changes` (boolean) и `changed_files` (число). При отключённом git MUST возвращаться 503.

#### Сценарий: Есть незакоммиченные изменения

- **WHEN** GET /api/git/status и git имеет modified/untracked/deleted файлы
- **THEN** возвращается `{ "has_changes": true, "changed_files": N }`

#### Сценарий: Нет изменений

- **WHEN** GET /api/git/status и рабочий каталог чист
- **THEN** возвращается `{ "has_changes": false, "changed_files": 0 }`

#### Сценарий: Git отключён

- **WHEN** GET /api/git/status и KB_GIT_DISABLED=true
- **THEN** возвращается 503

### Requirement: Git-коммит API

API MUST предоставлять эндпоинт `POST /api/git/commit` для коммита всех изменений с автогенерацией commit message через LLM. Тело MAY содержать `{ "message"?: string }`. При отсутствии message MUST вызываться LLM (OpenAI Responses API) для генерации conventional commit message на основе `git diff --stat`. При недоступности LLM MUST использоваться fallback `chore: manual commit via UI`. Выполняет git add -A, commit, push. При отключённом git — 503.

#### Сценарий: Автогенерация commit message

- **WHEN** POST /api/git/commit без message и есть изменения
- **THEN** LLM генерирует conventional commit message, выполняется commit+push, возвращается `{ message, committed: true }`

#### Сценарий: Ручной commit message

- **WHEN** POST /api/git/commit с `{ "message": "fix: typo" }`
- **THEN** используется указанное сообщение, LLM не вызывается

#### Сценарий: Нет изменений

- **WHEN** POST /api/git/commit и нет незакоммиченных изменений
- **THEN** возвращается `{ "committed": false, "message": "no changes to commit" }`
