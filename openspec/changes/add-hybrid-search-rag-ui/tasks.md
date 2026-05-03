## 1. Index Schema And Sync

- [ ] 1.1 Расширить SQLite schema/migrations для searchable text нод и чанков.
- [ ] 1.2 Добавить detection режима keyword index: `fts5`, `scan` или `disabled`.
- [ ] 1.3 Обновить `IndexStatus`, чтобы `GET /api/index/status` возвращал `keyword_index`.
- [ ] 1.4 Обновить SyncWorker: при индексации ноды записывать searchable text для node-level поиска.
- [ ] 1.5 Обновить SyncWorker: при индексации article chunks записывать searchable text для chunk-level поиска.
- [ ] 1.6 Обновить ManualRebuild/ClearAll так, чтобы очищались новые searchable tables.
- [ ] 1.7 Добавить unit-тесты schema migration, status и очистки searchable данных.

## 2. Keyword Search

- [ ] 2.1 Спроектировать Go-типы keyword candidates: node hit, chunk hit, match fields, raw score/rank.
- [ ] 2.2 Реализовать keyword/FTS поиск по node searchable text.
- [ ] 2.3 Реализовать keyword/FTS поиск по chunk searchable text.
- [ ] 2.4 Реализовать fallback scan при недоступном FTS5.
- [ ] 2.5 Добавить exact boosts для `title`, `path`, `aliases`, `keywords`.
- [ ] 2.6 Добавить snippets для chunk/body совпадений.
- [ ] 2.7 Покрыть keyword search unit-тестами: keyword, title, path, chunk content, fallback scan.

## 3. Hybrid Retrieval

- [ ] 3.1 Создать `RetrievalService` или эквивалентный слой в `internal/index`.
- [ ] 3.2 Определить request/options: query, mode, filters, topK/limit, source_paths.
- [ ] 3.3 Определить response model: ranked results, fragments, score, match_reasons, source_kinds.
- [ ] 3.4 Подключить keyword candidates, vector node candidates и vector chunk candidates к единому pipeline.
- [ ] 3.5 Реализовать fusion/ranking через RRF с boost для exact/keyword совпадений.
- [ ] 3.6 Реализовать фильтры по type, path/recursive, manual_processed и source_paths.
- [ ] 3.7 Реализовать chat relevance cutoff для слабых vector-only результатов.
- [ ] 3.8 Добавить unit-тесты fusion/ranking/cutoff/filtering на маленьких фикстурах.

## 4. REST API

- [ ] 4.1 Заменить заглушку `GET /api/search` или добавить `POST /api/search` согласно спецификации.
- [ ] 4.2 Реализовать валидацию request body для `POST /api/search`.
- [ ] 4.3 Реализовать JSON response гибридного поиска с карточками и fragments.
- [ ] 4.4 Обновить роутинг API для нового search endpoint.
- [ ] 4.5 Обновить `POST /api/chat`, чтобы он использовал `RetrievalService` вместо прямых `VectorSearch`/`ChunkSearch`.
- [ ] 4.6 Добавить поддержку `source_paths` в `POST /api/chat`.
- [ ] 4.7 Обновить SSE `sources` event: path, title, type, fragments.
- [ ] 4.8 Добавить API-тесты для `POST /api/search`: success, empty query, filters, service unavailable.
- [ ] 4.9 Обновить API-тесты для `POST /api/chat`: hybrid retrieval, source_paths, empty sources/cutoff.
- [ ] 4.10 Обновить API-тесты для `GET /api/index/status` с `keyword_index`.

## 5. Chat Context Assembly

- [ ] 5.1 Переписать сборку RAG context на основе ranked hybrid results.
- [ ] 5.2 Добавить форматирование fragments с heading/snippet/content для LLM context.
- [ ] 5.3 Сохранить общий token budget около 4000 токенов.
- [ ] 5.4 Реализовать корректное поведение при пустом/недостаточном контексте.
- [ ] 5.5 Добавить unit-тесты context assembly: links, notes, article chunks, duplicate node coverage, empty context.

## 6. Frontend API Client

- [ ] 6.1 Добавить TypeScript-типы для index status `keyword_index`.
- [ ] 6.2 Добавить `searchKnowledgeBase`/эквивалент для `POST /api/search`.
- [ ] 6.3 Расширить `ChatSource` типом, title, fragments.
- [ ] 6.4 Расширить `streamChat`, чтобы он мог отправлять `source_paths`.
- [ ] 6.5 Добавить unit-тесты/тесты service layer для нового search client и обновлённого chat parsing.

## 7. Search UI

- [ ] 7.1 Добавить route `/search` и вкладку «Поиск» в Navbar при доступности индекса.
- [ ] 7.2 Создать страницу поиска с input, loading/error/empty states.
- [ ] 7.3 Добавить фильтры по type, path/теме и manual_processed.
- [ ] 7.4 Отобразить результаты карточками: title, type, annotation, path, keywords, match reasons.
- [ ] 7.5 Отобразить article fragments: heading/snippet и источник совпадения.
- [ ] 7.6 Добавить переход по title карточки на страницу ноды с сохранением returnTo.
- [ ] 7.7 Добавить действие “Спросить по этим источникам” с передачей query/source_paths в чат.
- [ ] 7.8 Добавить frontend-тесты SearchPage: success, empty, unavailable, action to chat.

## 8. Chat UI

- [ ] 8.1 Обновить ChatPage для initial message/source hints из route state или query params.
- [ ] 8.2 Обновить отображение sources: title, type, path, fragments.
- [ ] 8.3 Добавить раскрытие найденного контекста/fragments.
- [ ] 8.4 Показать “недостаточно данных в базе” как нормальное состояние, не техническую ошибку.
- [ ] 8.5 Добавить frontend-тесты ChatPage для sources with fragments и source_paths flow.

## 9. Verification

- [ ] 9.1 Запустить `go test ./...`.
- [ ] 9.2 Запустить frontend tests (`npm test` или существующую команду проекта).
- [ ] 9.3 Запустить frontend build (`npm run build` в `web/`).
- [ ] 9.4 Запустить `openspec validate add-hybrid-search-rag-ui`.
- [ ] 9.5 Ручная проверка: rebuild индекса, поиск по точному keyword, semantic-only поиск, переход поиск → чат.
