## Purpose

MCP (Model Context Protocol) сервер для подключения чатботов (Claude, Cursor и др.) к базе знаний. Endpoint на том же сервере, что и API.

## Requirements

### Requirement: Endpoint /api/mcp

MCP MUST быть доступен по пути /api/mcp на том же HTTP-сервере, что и REST API.

#### Сценарий: Подключение MCP-клиента

- **WHEN** MCP-клиент подключается к /api/mcp
- **THEN** устанавливается соединение по протоколу MCP (SSE/WebSocket или HTTP)

### Requirement: Доступ к базе

MCP-сервер ДОЛЖЕН (SHALL) использовать KB_DATA_PATH для доступа к базе знаний.

#### Сценарий: Запрос контента через MCP

- **WHEN** чатбот запрашивает контент через MCP
- **THEN** сервер возвращает данные из базы по пути KB_DATA_PATH
