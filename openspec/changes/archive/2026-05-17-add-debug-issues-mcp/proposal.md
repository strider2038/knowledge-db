## Why

Поддержка продукта по конкретным кейсам сейчас требует ручного сбора контекста из UI, логов и состояния чата, что замедляет отладку и доработки. Нужен встроенный механизм, который сохраняет самодостаточные отладочные артефакты локально и дает агентам удобный машинный доступ к ним.

## What Changes

- Добавить сохранение сырых Telegram-сообщений в `KB_DATA_PATH/.kb/telegram-raw/` в формате NDJSON при включенном env-флаге.
- Добавить в web UI кнопку багрепорта на страницах узла, семантического поиска и чата с модальным вводом описания проблемы.
- Добавить серверный API для сохранения багрепортов в `KB_DATA_PATH/.kb/issues/...` в файловом формате Markdown + frontmatter.
- Добавить серверный API обновления статуса issue (`new`, `investigating`, `fixed`) для рабочего пайплайна сопровождения.
- Сохранять в багрепортах полный диагностический контекст страницы (для чата — полные сообщения и связанные источники).
- Добавить отдельный служебный MCP endpoint для чтения debug-артефактов (issues и Telegram raw) без смешивания с основным MCP.
- Добавить отдельный API-ключ для служебного MCP endpoint.
- Добавить автоматическую ротацию raw Telegram логов с TTL 14 дней.

## Capabilities

### New Capabilities
- `debug-issue-reporting`: Локальное файловое сохранение отладочных артефактов (issues и Telegram raw) и служебный MCP-доступ для агентов.

### Modified Capabilities
- Нет.

## Impact

- Backend:
  - `internal/bootstrap/config` (новые env-переменные)
  - `internal/telegram` (опциональный raw logging)
  - `internal/api` (endpoint сохранения issue)
  - `internal/mcp` или отдельный debug MCP handler/роутинг
- Frontend:
  - `web/src/pages/NodePage.tsx`
  - `web/src/pages/SearchPage.tsx`
  - `web/src/pages/ChatPage.tsx`
  - `web/src/services/api.ts`
- Документация:
  - `README.md`
  - `.env.example`
- Тесты:
  - API-тесты новых endpoint'ов
  - unit/integration тесты на запись файлов и debug MCP handlers
