## 1. Конфигурация и безопасность доступа MCP

- [x] 1.1 Добавить конфигурацию `KB_MCP_API_KEY` в bootstrap/config и логику отключения MCP при пустом/отсутствующем ключе.
- [x] 1.2 Реализовать auth-проверку Bearer-токена для `/api/mcp` с ответом `401` для отсутствующего/невалидного ключа.
- [x] 1.3 Обновить wiring маршрута `/api/mcp` в bootstrap с подключением новой auth-логики.

## 2. Реализация MCP сервера на go-sdk

- [x] 2.1 Подключить зависимость `github.com/modelcontextprotocol/go-sdk` и создать MCP server вместо заглушки.
- [x] 2.2 Реализовать tool `search_notes` с переиспользованием существующего retrieval/index слоя.
- [x] 2.3 Реализовать tool `semantic_search` с корректной обработкой режима отключённых эмбеддингов.

## 3. Тесты и валидация

- [x] 3.1 Добавить API/MCP-тесты на авторизацию `/api/mcp` (valid key, no auth, invalid key).
- [x] 3.2 Добавить тесты инструментов `search_notes` и `semantic_search` (включая semantic unavailable path).
- [x] 3.3 Прогнать `openspec validate mcp-api-key-auth` и проверить `openspec status --change mcp-api-key-auth`.

## 4. Tool для чтения заметки

- [x] 4.1 Реализовать MCP tool `get_note` для чтения узла по `path` с опциональным `max_chars`.
- [x] 4.2 Добавить тесты `get_note` (успех, валидация path, not found, truncation).
