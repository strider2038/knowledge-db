## Why

Пользователь экспортирует чат «Избранное» из Telegram в JSON и хочет массово обработать записи через UI — последовательно принять или отклонить каждую. Процесс может растягиваться на дни; сессия должна сохраняться на backend и восстанавливаться после перезагрузки.

## What Changes

- **Новый режим импорта** на странице «Добавить»: вкладка «Импорт из Telegram» с загрузкой JSON, пошаговой обработкой записей (принять/отклонить), индикатором прогресса.
- **Backend API** для импорт-сессий: создание сессии из JSON, получение состояния, действия accept/reject с вызовом ingestion.
- **Хранение сессий** в файлах: `{KB_UPLOADS_DIR}/telegram-import-sessions/{session_id}.json`. Путь задаётся переменной окружения KB_UPLOADS_DIR.
- **Парсинг Telegram export** (формат одного чата): извлечение text из `text`/`text_entities`, source_author (forwarded_from || saved_from || from), source_url из link-сущностей, caption для медиа. Сортировка по date_unixtime desc.
- **Расширение web API** `ingestText`: передача source_url и source_author в POST /api/ingest.

## Capabilities

### New Capabilities

- `telegram-import-bulk`: массовая обработка экспортированного чата Telegram — парсинг JSON, backend-сессии, API accept/reject, UI с табами и прогрессом.

### Modified Capabilities

- `rest-api`: новые эндпоинты POST /api/import/telegram, GET/POST /api/import/telegram/session/:id, конфигурация KB_UPLOADS_DIR.
- `webapp`: раздел «Добавить» — табы «Вручную» | «Импорт из Telegram», передача source_url/source_author в ingest.

## Impact

- **internal/api**: новые handlers для import API.
- **internal/import**: пакет парсинга Telegram export JSON.
- **web/src/pages/AddPage.tsx**: табы, компонент импорта.
- **web/src/services/api.ts**: расширение ingestText (source_url, source_author).
- **Конфигурация**: KB_UPLOADS_DIR (обязательна для импорта).
