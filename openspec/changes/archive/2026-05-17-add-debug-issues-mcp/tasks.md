## 1. Конфигурация и файловые хранилища debug-артефактов

- [x] 1.1 Добавить env-переменные `KB_TELEGRAM_RAW_LOG_ENABLED` и `KB_MCP_DEBUG_API_KEY` в конфиг, `.env.example` и `README.md`
- [x] 1.2 Реализовать файловый writer для issue-репортов в `KB_DATA_PATH/.kb/issues` (Markdown + frontmatter, иерархия по дате)
- [x] 1.3 Реализовать файловый writer для Telegram raw логов в `KB_DATA_PATH/.kb/telegram-raw/*.ndjson` с append-семантикой и TTL 14 дней

## 2. Backend API и интеграция Telegram

- [x] 2.1 Добавить API endpoint создания issue-репорта из web UI с валидацией входного payload
- [x] 2.2 Добавить API endpoint обновления статуса issue (`new`, `investigating`, `fixed`) с валидацией допустимых значений статуса
- [x] 2.3 Интегрировать raw logging в `internal/telegram/bot.go` с учетом флага и безопасной деградации при ошибках записи
- [x] 2.4 Интегрировать регулярную очистку raw Telegram файлов старше 14 дней
- [x] 2.5 Написать API-тесты для endpoint'ов создания/обновления issue и unit-тесты для файловых writers/TTL очистки

## 3. Debug MCP endpoint

- [x] 3.1 Добавить отдельный debug MCP handler/endpoint с авторизацией по `KB_MCP_DEBUG_API_KEY`
- [x] 3.2 Реализовать debug tools: список issues, чтение issue, чтение последних Telegram raw записей
- [x] 3.3 Добавить тесты debug MCP handler/tool-методов (успех, выключенный endpoint, невалидный ключ)

## 4. Web UI багрепорты

- [x] 4.1 Добавить кнопку и модалку багрепорта на NodePage с отправкой контекста узла
- [x] 4.2 Добавить кнопку и модалку багрепорта на SearchPage с отправкой query/filters/results/meta
- [x] 4.3 Добавить кнопку и модалку багрепорта на ChatPage с отправкой полного контекста чата (полные сообщения + sources)
- [x] 4.4 Добавить frontend тесты/обновить существующие тесты страниц и API-клиента для нового endpoint

## 5. Проверка и готовность к применению

- [x] 5.1 Прогнать релевантные backend и frontend тесты для новых сценариев
- [x] 5.2 Проверить, что debug-данные сохраняются только в `.kb/...` и не влияют на основное дерево узлов
- [x] 5.3 Подготовить короткую инструкцию для агента по использованию debug MCP tools в рамках change
