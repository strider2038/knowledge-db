## Purpose

Уточнить требования к доступу MCP endpoint при включённой опциональной сессионной авторизации.

## Requirements

## MODIFIED Requirements

### Requirement: Endpoint /api/mcp

MCP MUST быть доступен по пути /api/mcp на том же HTTP-сервере, что и REST API. При включённой авторизации (`KB_LOGIN` и `KB_PASSWORD` заданы) endpoint `/api/mcp` MUST требовать валидную сессионную cookie; при отсутствии или невалидности сессии сервер SHALL возвращать `401 Unauthorized`.

#### Сценарий: Подключение MCP-клиента

- **WHEN** MCP-клиент подключается к /api/mcp
- **THEN** устанавливается соединение по протоколу MCP (SSE/WebSocket или HTTP)

#### Сценарий: Подключение без сессии при включённой авторизации

- **WHEN** авторизация включена и MCP-клиент подключается к /api/mcp без валидной сессии
- **THEN** сервер возвращает `401 Unauthorized`
