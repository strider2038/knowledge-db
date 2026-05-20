## 1. KB Store и идентичность

- [ ] 1.1 Добавить генерацию UUID v7 в CreateNode/CreateTranslationFile (gofrs/uuid v7)
- [ ] 1.2 Реализовать GetNodeByID, индекс id→path при обходе или кэш в Store
- [ ] 1.3 Обновить MoveNode: сохранение id в frontmatter, обновление только path
- [ ] 1.4 Добавить translation_of_id при программном создании перевода
- [ ] 1.5 Расширить validator: обязательный id, уникальность id, формат UUID
- [ ] 1.6 Unit-тесты kb.Store: create id, move id stable, GetNodeByID

## 2. CLI миграция

- [ ] 2.1 Подкоманда `kb-cli migrate-node-ids` с `--dry-run` и `--path`
- [ ] 2.2 Отчёт о конфликтах id; идемпотентность повторного запуска
- [ ] 2.3 Тесты CLI migrate (afero FS)

## 3. Embedding index (SQLite)

- [ ] 3.1 Миграция схемы: node_id PK, path UNIQUE, chunks/node_search на node_id
- [ ] 3.2 Таблица node_source_urls + upsert/delete при sync
- [ ] 3.3 UpdateNodePath при move; убрать delete+recreate по старому path
- [ ] 3.4 SyncWorker: чтение id из frontmatter, skip/warn без id
- [ ] 3.5 FindBySourceURL для ingestion
- [ ] 3.6 Тесты sqlite store: upsert, move path, source_url lookup

## 4. Ingestion dedup

- [ ] 4.1 saveNode: resolve create vs update (node_id, source_url)
- [ ] 4.2 UpdateNode (метаданные + body) без смены path при dedup по URL
- [ ] 4.3 Тесты pipeline: повторный URL → один файл, тот же id

## 5. REST API

- [ ] 5.1 Поле `id` в JSON-ответах узла и списка
- [ ] 5.2 `GET /api/nodes/by-id/{id}`
- [ ] 5.3 Move handler: триггер index update по node_id
- [ ] 5.4 API-тесты: by-id, move id stable, list with id

## 6. MCP

- [ ] 6.1 get_note: id в ответе; опциональный входной параметр id
- [ ] 6.2 Тесты mcp handler

## 7. Frontend

- [ ] 7.1 Тип Node/NodeListItem: поле `id` в api.ts
- [ ] 7.2 При необходимости отображение id (debug/copy) — минимально

## 8. Документация и финализация

- [ ] 8.1 README / .env.example: шаги migrate-node-ids + reindex после апгрейда
- [ ] 8.2 Убрать пункт todo «суррогатный id» или отметить выполненным
- [ ] 8.3 `openspec validate add-node-uuid-v7`
