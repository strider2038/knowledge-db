## 1. Дедупликация ingestion

- [x] 1.1 В `resolveExistingNode` (`internal/ingestion/save_node_dedup.go`) пропускать lookup по `source_url`, если `result.Type` пустой или равен `note`
- [x] 1.2 Добавить `clog.Info` при пропуске дедупа по URL из-за типа контента
- [x] 1.3 Вынести проверку «дедуп по URL разрешён для типа» в небольшую функцию (например `ingestTypeAllowsSourceURLDedup`) для читаемости и unit-тестов

## 2. Тесты

- [x] 2.1 Добавить `TestPipelineIngester_IngestText_WhenNoteWithDuplicateSourceURL_ExpectNewNode` — существующий узел с URL, новая заметка с тем же URL → два узла, старый контент не меняется
- [x] 2.2 Убедиться, что `TestPipelineIngester_IngestURL_WhenDuplicateSourceURL_ExpectSameIDAndPath` по-прежнему проходит (article/link дедуп сохранён)
- [x] 2.3 Запустить `go test ./internal/ingestion/... -race` и `golangci-lint run ./internal/ingestion/...`

## 3. Верификация change

- [x] 3.1 Выполнить `openspec validate fix-ingest-source-url-dedup`
- [x] 3.2 Отметить выполненные пункты в `tasks.md` после реализации
