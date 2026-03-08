# Tasks: Сохранение автора и ссылки на источник

## 1. Интерфейс и типы ingestion

- [x] 1.1 Добавить структуру `IngestRequest` с полями Text, SourceURL, SourceAuthor в `internal/ingestion`
- [x] 1.2 Изменить интерфейс Ingester: `IngestText(ctx, IngestRequest)` вместо `IngestText(ctx, string)`

## 2. Pipeline и LLM

- [x] 2.1 Добавить в `ProcessInput` опциональные поля SourceURL, SourceAuthor
- [x] 2.2 Обновить `buildProcessInput`: принимать sourceURL, sourceAuthor; при наличии добавлять блок «Метаданные источника» в начало текста
- [x] 2.3 Обновить `PipelineIngester.IngestText`: принимать IngestRequest, передавать SourceURL/SourceAuthor в buildProcessInput
- [x] 2.4 Обновить `PipelineIngester.IngestURL`: передавать url и fetchResult.Author в buildProcessInput
- [x] 2.5 Добавить source_author в schema tool create_node в `buildTools`
- [x] 2.6 Добавить SourceAuthor в ProcessResult и parseCreateNodeArgs
- [x] 2.7 В orchestrator: при слиянии с fetch cache подставлять SourceAuthor и SourceDate из FetchResult, если LLM не вернул
- [x] 2.8 В saveNode: добавлять source_author в frontmatter при наличии

## 3. Telegram-бот

- [x] 3.1 Реализовать `parseForwardOrigin(raw json.RawMessage) (sourceURL, sourceAuthor string)` для типов channel, user, chat, hidden_user
- [x] 3.2 Обновить processIngest: принимать sourceURL, sourceAuthor; вызывать IngestText с IngestRequest
- [x] 3.3 В handleUpdate: при пересланном сообщении вызывать parseForwardOrigin, передавать результат в processIngest
- [x] 3.4 В handleUpdate: при reply на пересланное брать forward_origin из reply_to_message

## 4. Заглушки и API

- [x] 4.1 Обновить StubIngester: IngestText(ctx, IngestRequest)
- [x] 4.2 Обновить API handler ingest: парсить source_url, source_author из JSON (опционально); вызывать IngestText(IngestRequest{...})

## 5. Тесты

- [x] 5.1 Обновить mock Ingester в internal/telegram/bot_test.go под новую сигнатуру
- [x] 5.2 Обновить mock Ingester в internal/api/handlers_test.go
- [x] 5.3 Обновить internal/ingestion/pipeline_test.go: вызовы IngestText с IngestRequest
- [x] 5.4 Добавить тест parseForwardOrigin для типов channel, user, hidden_user
- [x] 5.5 Добавить тест: пересланное сообщение → IngestText вызывается с SourceURL и SourceAuthor
- [x] 5.6 Добавить тест orchestrator: подстановка SourceAuthor из FetchResult при create_node
- [x] 5.7 API тест: POST /api/ingest с source_url, source_author в теле (если endpoint меняется)
