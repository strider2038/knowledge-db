## 1. KB Store и идентичность

- [x] 1.1 Добавить генерацию UUID v7 в CreateNode/CreateTranslationFile (gofrs/uuid v7)
- [x] 1.2 Реализовать GetNodeByID, индекс id→path при обходе или кэш в Store
- [x] 1.3 Обновить MoveNode: сохранение id в frontmatter, обновление только path
- [x] 1.4 Добавить translation_of_id при программном создании перевода
- [x] 1.5 Расширить validator: обязательный id, уникальность id, формат UUID
- [x] 1.6 Unit-тесты kb.Store: create id, move id stable, GetNodeByID

## 2. CLI миграция

- [x] 2.1 Подкоманда `kb-cli migrate-node-ids` с `--dry-run` и `--path`
- [x] 2.2 Отчёт о конфликтах id; идемпотентность повторного запуска
- [x] 2.3 Тесты CLI migrate (afero FS)

## 3. Embedding index (SQLite)

- [x] 3.1 Миграция схемы: node_id PK, path UNIQUE, chunks/node_search на node_id
- [x] 3.2 Таблица node_source_urls + upsert/delete при sync
- [x] 3.3 UpdateNodePath при move; убрать delete+recreate по старому path
- [x] 3.4 SyncWorker: чтение id из frontmatter, skip/warn без id
- [x] 3.5 FindBySourceURL для ingestion
- [x] 3.6 Тесты sqlite store: upsert, move path, source_url lookup

## 4. Ingestion dedup

- [x] 4.1 saveNode: resolve create vs update (node_id, source_url)
- [x] 4.2 UpdateNode (метаданные + body) без смены path при dedup по URL
- [x] 4.3 Тесты pipeline: повторный URL → один файл, тот же id

## 5. REST API

- [x] 5.1 Поле `id` в JSON-ответах узла и списка
- [x] 5.2 `GET /api/nodes/by-id/{id}`
- [x] 5.3 Move handler: триггер index update по node_id
- [x] 5.4 API-тесты: by-id, move id stable, list with id

## 6. MCP

- [x] 6.1 get_note: id в ответе; опциональный входной параметр id
- [x] 6.2 Тесты mcp handler

## 7. Frontend

- [x] 7.1 Тип Node/NodeListItem: поле `id` в api.ts
- [x] 7.2 При необходимости отображение id (debug/copy) — минимально

## 8. Документация и финализация

- [x] 8.1 README / .env.example: шаги migrate-node-ids + reindex после апгрейда
- [x] 8.2 Убрать пункт todo «суррогатный id» или отметить выполненным
- [x] 8.3 `openspec validate add-node-uuid-v7`
