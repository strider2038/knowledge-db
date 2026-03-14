## Purpose

MCP (Model Context Protocol) сервер для подключения чатботов (Claude, Cursor и др.) к базе знаний. Endpoint на том же сервере, что и API.

## Requirements

### Requirement: Endpoint /api/mcp

MCP MUST быть доступен по пути /api/mcp на том же HTTP-сервере, что и REST API. При включённой авторизации (`KB_LOGIN` и `KB_PASSWORD` заданы) endpoint `/api/mcp` MUST требовать валидную сессионную cookie; при отсутствии или невалидности сессии сервер SHALL возвращать `401 Unauthorized`.

#### Сценарий: Подключение MCP-клиента

- **WHEN** MCP-клиент подключается к /api/mcp
- **THEN** устанавливается соединение по протоколу MCP (SSE/WebSocket или HTTP)

#### Сценарий: Подключение без сессии при включённой авторизации

- **WHEN** авторизация включена и MCP-клиент подключается к /api/mcp без валидной сессии
- **THEN** сервер возвращает `401 Unauthorized`

### Requirement: Доступ к базе

MCP-сервер ДОЛЖЕН (SHALL) использовать KB_DATA_PATH для доступа к базе знаний.

#### Сценарий: Запрос контента через MCP

- **WHEN** чатбот запрашивает контент через MCP
- **THEN** сервер возвращает данные из базы по пути KB_DATA_PATH
