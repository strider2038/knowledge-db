# Инструкция для агента: debug MCP tools

Endpoint: `POST/GET /api/mcp/debug`  
Auth: `Authorization: Bearer <KB_MCP_DEBUG_API_KEY>`

## Доступные tools

1. `debug_list_issues`
- Назначение: получить список последних issue-репортов.
- Вход: `{ "limit": 20 }` (опционально).

2. `debug_get_issue`
- Назначение: прочитать полный issue по `id` (включая context/body).
- Вход: `{ "id": "issue-..." }`.

3. `debug_get_telegram_raw`
- Назначение: получить последние raw Telegram записи из `.kb/telegram-raw`.
- Вход: `{ "limit": 50 }` (опционально).

## Рекомендуемый flow

1. Вызвать `debug_list_issues` и выбрать нужный `id`.
2. Вызвать `debug_get_issue` для полного контекста кейса.
3. При необходимости корреляции с Telegram вызвать `debug_get_telegram_raw`.
4. Для смены статуса issue использовать HTTP API:
   `PATCH /api/debug/issues/{id}` с body `{"status":"investigating|fixed|new"}`.
