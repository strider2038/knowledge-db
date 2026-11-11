# Solution Structure

## Deployable Root

`/home/agent/work/knowledge-db` — единый бинарь `kb` и фронтенд `web/`.

## Added / Modified / Removed Tree

```text
internal/bootstrap/config/config.go          # поля KB_NODE_AGENT*
internal/bootstrap/config/node_agent_test.go # валидация
internal/bootstrap/bootstrap.go              # выбор раннера, вызов валидации
internal/api/cursor_agent.go                 # раннер Cursor, Available()
internal/api/opencode_agent.go               # раннер OpenCode, конфиг, JSONL
internal/api/opencode_agent_test.go          # argv, права, парсер, ошибка JSONL
internal/api/node_normalization.go           # Available() вместо type assert
internal/api/node_agent_edit.go              # то же
internal/api/jobs.go                         # ошибка недоступности opencode
Dockerfile                                   # установка opencode 2.2.7
.env.example
README.md
web/src/components/NodeAgentEditDialog.tsx
web/src/components/NodeAgentEditDialog.test.tsx
```

Новый пакет не вводится: раннер остаётся в `internal/api` рядом с джобами.

## Interface-to-Implementation Mapping

| Route / interface / UI surface | Owning module | Entry point | Notes |
|--------------------------------|---------------|-------------|-------|
| `POST /api/nodes/normalize` | `internal/api` | `PostNodeNormalize` | Общий раннер |
| `POST /api/nodes/agent-edit` | `internal/api` | `PostNodeAgentEdit` | Тот же раннер |
| `KB_NODE_AGENT*` | `internal/bootstrap/config` | `ValidateNodeAgent` | Ошибка до bind |
| Модалка agent-edit | `web/src/components/NodeAgentEditDialog.tsx` | `DialogDescription` | Без имени поставщика |
| Образ | `Dockerfile` | stage runtime | `cursor-agent` и `opencode` |

## Dependency and Ownership Boundaries

`internal/api` не импортирует bootstrap. Конфиг агента передаётся в конструктор раннера из `bootstrap.Run`. Промпты нормализации и правки остаются в своих файлах. Парсер Cursor не вызывается для OpenCode.

## Temporary Wiring / Stubs

Нет. Тесты подменяют раннер моком, как сейчас. Проверка 503 использует реальный раннер и отсутствие бинаря в PATH теста.
