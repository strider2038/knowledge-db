## Purpose

MCP (Model Context Protocol) сервер для подключения чатботов (Claude, Cursor и др.) к базе знаний. Endpoint на том же сервере, что и API.
## Requirements
### Requirement: Endpoint /api/mcp

MCP MUST быть доступен по пути `/api/mcp` на том же HTTP-сервере, что и REST API. Endpoint `/api/mcp` MUST требовать Bearer-токен, равный значению `KB_MCP_API_KEY`. При отсутствии токена, неверном формате `Authorization` или невалидном ключе сервер SHALL возвращать `401 Unauthorized`. Реализация MCP endpoint MUST использовать библиотеку `github.com/modelcontextprotocol/go-sdk`.

#### Scenario: Подключение MCP-клиента с валидным API-ключом

- **WHEN** MCP-клиент подключается к `/api/mcp` с заголовком `Authorization: Bearer <KB_MCP_API_KEY>`
- **THEN** сервер устанавливает MCP-соединение и обрабатывает MCP-запросы

#### Scenario: Подключение без заголовка Authorization

- **WHEN** MCP-клиент подключается к `/api/mcp` без заголовка `Authorization`
- **THEN** сервер возвращает `401 Unauthorized`

#### Scenario: Подключение с невалидным ключом

- **WHEN** MCP-клиент подключается к `/api/mcp` с Bearer-токеном, не совпадающим с `KB_MCP_API_KEY`
- **THEN** сервер возвращает `401 Unauthorized`

#### Scenario: MCP отключен при пустом ключе

- **WHEN** `KB_MCP_API_KEY` не задан или пустой
- **THEN** MCP endpoint не работает (маршрут `/api/mcp` не должен обслуживаться сервером)

### Requirement: Доступ к базе

MCP-сервер ДОЛЖЕН (SHALL) использовать `KB_DATA_PATH` для доступа к базе знаний и MUST использовать существующий индекс/retrieval-слой проекта для операций поиска.

#### Scenario: Запрос контента через MCP

- **WHEN** MCP-клиент вызывает MCP tool для поиска по базе знаний
- **THEN** сервер возвращает данные из базы по пути `KB_DATA_PATH`

### Requirement: Идентификатор узла в MCP get_note

Инструмент чтения узла (get_note) MUST возвращать в JSON-ответе поле `id` (UUID узла) вместе с `path`, `title` и остальными полями. Входной параметр `path` MUST оставаться основным способом адресации; опционально инструмент MAY принимать `id` для чтения узла по стабильному идентификатору.

#### Scenario: get_note по path возвращает id

- **WHEN** MCP get_note вызывается с path существующего узла
- **THEN** ответ содержит непустые поля `id` и `path`

#### Scenario: get_note по id

- **WHEN** MCP get_note вызывается с параметром id существующего узла
- **THEN** возвращается контент узла с актуальным path

