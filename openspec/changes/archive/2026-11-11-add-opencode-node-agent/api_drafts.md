# API Drafts

## Conventions Consulted

- Existing contract/example: `POST /api/nodes/normalize`, `POST /api/nodes/agent-edit` с `path` в JSON-теле; `GET /api/node-normalization/{id}` и `/logs`, `GET /api/node-agent-edit/{id}` и `/logs`.
- Project API skill/document: `.agents/skills/api-conventions/SKILL.md` — POST-action мутации, REST GET, snake_case.
- Compatibility baseline: те же маршруты и поля ответа. Меняется только диагностика 503 выбранного агента.

## New or Changed Interfaces

Новых маршрутов нет. Ниже — сохранённый контракт и изменённая ошибка.

### `POST /api/nodes/normalize`

**Purpose:** запуск нормализации узла выбранным агентом.

**Authorization / tenant scope:** как у остальных node-мутаций (сессия, если auth включена).

**Compatibility / versioning:** тело и успешный ответ без изменений.

#### Request / Input

```json
{ "path": "topic/note.md" }
```

#### Response / Output

```json
{
  "id": "job-id",
  "node_path": "topic/note.md",
  "status": "running",
  "stage": "normalize",
  "started_at": "2026-11-11T12:00:00Z",
  "sync_done": false,
  "normalize_ok": false
}
```

#### Errors and Edge Cases

| Condition | Contract |
|-----------|----------|
| Пустой `path` | 400 |
| Узел не найден | 404 |
| Нормализация этого path уже `running` | 409, без второй операции |
| `KB_NODE_AGENT` пустой или `cursor`, нет `cursor-agent` | 503, тело содержит `cursor-agent not found in PATH` |
| `KB_NODE_AGENT=opencode`, нет `opencode` | 503, тело называет `opencode`, операция не создаётся |

#### Idempotency / Pagination / Streaming

Повтор во время running — конфликт, как сейчас. Логи стримятся опросом `GET /api/node-normalization/{id}/logs?after=`.

### `POST /api/nodes/agent-edit`

**Purpose:** правка узла по инструкции тем же агентом.

**Authorization / tenant scope:** как у нормализации.

**Compatibility / versioning:** тело `{ "path", "instruction" }` и поля операции без изменений.

#### Request / Input

```json
{
  "path": "topic/note.md",
  "instruction": "Сократи вступление"
}
```

#### Response / Output

```json
{
  "id": "job-id",
  "node_path": "topic/note.md",
  "status": "running",
  "stage": "edit",
  "started_at": "2026-11-11T12:00:00Z",
  "sync_done": false,
  "edit_ok": false
}
```

#### Errors and Edge Cases

| Condition | Contract |
|-----------|----------|
| Пустой `path` или пустая `instruction` | 400 |
| `instruction` длиннее 16 KiB | 400 |
| Узел не найден | 404 |
| Agent-edit этого path уже `running` | 409 |
| Нет бинаря выбранного агента | 503, как у нормализации |

#### Idempotency / Pagination / Streaming

Как у нормализации. Логи: `GET /api/node-agent-edit/{id}/logs?after=`. Записи: `offset`, `stream`, `text`, `timestamp`.

### Конфигурация процесса, не HTTP

`KB_NODE_AGENT`, `KB_NODE_AGENT_MODEL`, `KB_NODE_AGENT_API_URL`, `KB_NODE_AGENT_API_KEY`, `KB_NODE_AGENT_STATE_DIR`, `OPENCODE_API_KEY` читаются при старте процесса. Ошибка валидации не отдаётся HTTP-клиенту: `kb serve` завершается до bind.

## Not applicable

Нового публичного JSON-поля и нового маршрута нет. Причина отдельной секции ошибок выше — сохранившийся 503 меняет текст для режима `opencode`.
