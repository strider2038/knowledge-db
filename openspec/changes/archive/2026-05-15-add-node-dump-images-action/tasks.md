## 1. Backend API и оркестрация dump images

- [x] 1.1 Добавить endpoint запуска `POST /api/nodes/{path}/dump-images` и endpoint статуса операции.
- [x] 1.2 Реализовать серверную операцию `dump images` для одного узла с блокировкой повторного запуска для активного path.
- [x] 1.3 Добавить буфер логов операции (`stdout/stderr/system`, offset/timestamp) и endpoint `GET /api/node-dump-images/{id}/logs?after=`.
- [x] 1.4 Реализовать post-step `sync` после успешного `dump images` и возврат итогового статуса (`success` / `sync error`).
- [x] 1.5 Добавить/обновить API-тесты для запуска, статуса, логов и ошибок валидации.

## 2. UI действие Dump images и панель логов

- [x] 2.1 Добавить кнопку «Выгрузить изображения» в Action Bar страницы узла с корректными правилами видимости (только article).
- [x] 2.2 Подключить запуск операции, состояния loading/disabled и обработку финальных статусов/ошибок.
- [x] 2.3 Добавить нижнюю collapsible панель логов `dump images` с инкрементальным polling по `after`.
- [x] 2.4 Добавить/обновить frontend тесты для кнопки и лог-панели `dump images`.

## 3. Проверка и документация

- [x] 3.1 Проверить совместимость новых контрактов с существующей моделью операций нормализации (без регрессии по UX).
- [x] 3.2 Прогнать `openspec validate add-node-dump-images-action` и `openspec status --change add-node-dump-images-action`.
