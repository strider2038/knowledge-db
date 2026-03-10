## 1. Конфигурация и парсинг

- [x] 1.1 Добавить KB_UPLOADS_DIR в Config (internal/bootstrap/config), передавать в Handler
- [x] 1.2 Создать internal/import/telegram: ParseChat(json) — парсинг одного чата, извлечение text, source_author, source_url из Message, сортировка по date_unixtime desc
- [x] 1.3 Unit-тесты для парсера (text как string и массив, source_author приоритет, link→Markdown)

## 2. Backend: хранение сессий

- [x] 2.1 Создать internal/import/session: SessionStore с методами Create, Get, Accept, Reject; хранение в {KB_UPLOADS_DIR}/telegram-import-sessions/{id}.json
- [x] 2.2 При отсутствии KB_UPLOADS_DIR — методы возвращают ошибку «import not configured»

## 3. Backend: API

- [x] 3.1 Добавить POST /api/import/telegram (body: JSON) — парсинг, создание сессии, ответ { session_id, total, current_item }
- [x] 3.2 Добавить GET /api/import/telegram/session/:id — состояние сессии
- [x] 3.3 Добавить POST /api/import/telegram/session/:id/accept (body: type_hint?) — ingest, advance
- [x] 3.4 Добавить POST /api/import/telegram/session/:id/reject — reject, advance
- [x] 3.5 API-тесты для всех import endpoints (с моком SessionStore или тестовой директорией)

## 4. Web: API и страница «Добавить»

- [x] 4.1 Расширить ingestText в api.ts: опциональные source_url, source_author
- [x] 4.2 Добавить функции createImportSession, getImportSession, acceptImportItem, rejectImportItem в api.ts
- [x] 4.3 Добавить табы «Вручную» | «Импорт из Telegram» на AddPage (Tabs из shadcn/ui или аналог)
- [x] 4.4 Вкладка «Вручную» — текущая форма (без изменений логики)
- [x] 4.5 Вкладка «Импорт из Telegram»: file input, загрузка JSON → createImportSession, сохранение session_id в localStorage, отображение прогресса и текущей записи, type hint, кнопки «Отклонить» / «Принять»
- [x] 4.6 При открытии вкладки «Импорт» — проверка session_id в localStorage; при наличии — GET session и отображение текущей записи (восстановление после перезагрузки страницы)
- [x] 4.7 Кнопка «Начать заново» — очистка session_id, сброс UI к загрузке файла
