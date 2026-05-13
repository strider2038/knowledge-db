## Why

Текущий endpoint `/api/mcp` в проекте является заглушкой и не позволяет подключать внешние кодинговые агенты и чатботы к базе знаний. Для персонального offline-first сценария нужна минимальная, предсказуемая и простая схема доступа без отдельного контура управления ключами.

## What Changes

- Реализовать рабочий MCP endpoint на `/api/mcp` для подключения внешних MCP-клиентов.
- Ввести аутентификацию MCP-запросов через секрет из окружения `KB_MCP_API_KEY`.
- Если `KB_MCP_API_KEY` пустой или не задан, считать MCP отключенным (endpoint не работает).
- Возвращать `401 Unauthorized` при отсутствии/невалидности Bearer-токена.
- Реализовать MCP-сервер на базе `github.com/modelcontextprotocol/go-sdk`.
- Добавить MCP tools для поиска по заметкам, семантического поиска и чтения заметки по path (`get_note`) с переиспользованием существующего retrieval/index и kb-слоя.
- Зафиксировать поведение при отключённых эмбеддингах: keyword-поиск доступен, semantic tool возвращает понятную ошибку недоступности.
- Не добавлять поддержку `KB_MCP_API_KEYS` в рамках этого change.

## Capabilities

### New Capabilities

- `mcp-search-tools`: MCP инструменты для keyword/hybrid, semantic поиска и чтения узла по path через существующий индекс и kb-слой.

### Modified Capabilities

- `mcp-server`: изменение требований доступа к `/api/mcp` с сессионной cookie-модели на Bearer API key из `KB_MCP_API_KEY`, а также конкретизация реализации через go-sdk.

## Impact

- Backend:
  - `internal/mcp/*` — реализация MCP server, auth и tool handlers.
  - `internal/bootstrap/config/*` — чтение и валидация `KB_MCP_API_KEY`.
  - `internal/bootstrap/bootstrap.go` — wiring MCP handler с зависимостями поиска/индекса.
  - `internal/index/*` — переиспользование RetrievalService для MCP tools.
- Спецификации OpenSpec:
  - модификация capability `mcp-server`;
  - добавление capability `mcp-search-tools`.
- Зависимости:
  - новая библиотека `github.com/modelcontextprotocol/go-sdk`.
- Тесты:
  - API/MCP-тесты на авторизацию и доступность tools;
  - тесты поведения semantic tool при выключенных эмбеддингах.
