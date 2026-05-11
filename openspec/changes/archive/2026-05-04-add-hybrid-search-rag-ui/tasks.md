## 1. Index Schema And Sync

- [x] 1.1 Расширить SQLite schema/migrations для searchable text нод и чанков.
- [x] 1.2 Добавить detection режима keyword index: `fts5`, `scan` или `disabled`.
- [x] 1.3 Обновить `IndexStatus`, чтобы `GET /api/index/status` возвращал `keyword_index`.
- [x] 1.4 Обновить SyncWorker: при индексации ноды записывать searchable text для node-level поиска.
- [x] 1.5 Обновить SyncWorker: при индексации article chunks записывать searchable text для chunk-level поиска.
- [x] 1.6 Обновить ManualRebuild/ClearAll так, чтобы очищались новые searchable tables.
- [x] 1.7 Добавить unit-тесты schema migration, status и очистки searchable данных.

## 2. Keyword Search

- [x] 2.1 Спроектировать Go-типы keyword candidates: node hit, chunk hit, match fields, raw score/rank.
- [x] 2.2 Реализовать keyword/FTS поиск по node searchable text.
- [x] 2.3 Реализовать keyword/FTS поиск по chunk searchable text.
- [x] 2.4 Реализовать fallback scan при недоступном FTS5.
- [x] 2.5 Добавить exact boosts для `title`, `path`, `aliases`, `keywords`.
- [x] 2.6 Добавить snippets для chunk/body совпадений.
- [x] 2.7 Покрыть keyword search unit-тестами: keyword, title, path, chunk content, fallback scan.

## 3. Hybrid Retrieval

- [x] 3.1 Создать `RetrievalService` или эквивалентный слой в `internal/index`.
- [x] 3.2 Определить request/options: query, mode, filters, topK/limit, source_paths.
- [x] 3.3 Определить response model: ranked results, fragments, score, match_reasons, source_kinds.
- [x] 3.4 Подключить keyword candidates, vector node candidates и vector chunk candidates к единому pipeline.
- [x] 3.5 Реализовать fusion/ranking через RRF с boost для exact/keyword совпадений.
- [x] 3.6 Реализовать фильтры по type, path/recursive, manual_processed и source_paths.
- [x] 3.7 Реализовать chat relevance cutoff для слабых vector-only результатов.
- [x] 3.8 Добавить unit-тесты fusion/ranking/cutoff/filtering на маленьких фикстурах.
- [x] 3.9 Добавить нормализацию keyword query со стоп-словами и fallback на исходные токены.
- [x] 3.10 Добавить token-level exact boost и match reason `exact_token`.
- [x] 3.11 Ограничить вклад нескольких vector chunks одной ноды в итоговый score.
- [x] 3.12 Исправить UTF-8 safe chunk/snippet slicing для кириллицы и других многобайтных символов.
- [x] 3.13 Добавить LLM query rewrite для search mode с fallback на исходный запрос.
- [x] 3.14 Добавить vocabulary hints selection с параметрами лимитов слов/терминов.

## 4. REST API

- [x] 4.1 Заменить заглушку `GET /api/search` или добавить `POST /api/search` согласно спецификации.
- [x] 4.2 Реализовать валидацию request body для `POST /api/search`.
- [x] 4.3 Реализовать JSON response гибридного поиска с карточками и fragments.
- [x] 4.4 Обновить роутинг API для нового search endpoint.
- [x] 4.5 Обновить `POST /api/chat`, чтобы он использовал `RetrievalService` вместо прямых `VectorSearch`/`ChunkSearch`.
- [x] 4.6 Добавить поддержку `source_paths` в `POST /api/chat`.
- [x] 4.7 Обновить SSE `sources` event: path, title, type, fragments.
- [x] 4.8 Добавить API-тесты для `POST /api/search`: success, empty query, filters, service unavailable.
- [x] 4.9 Обновить API-тесты для `POST /api/chat`: hybrid retrieval, source_paths, empty sources/cutoff.
- [x] 4.10 Обновить API-тесты для `GET /api/index/status` с `keyword_index`.
- [x] 4.11 Добавить `meta.query_rewrite` в `POST /api/search`.
- [x] 4.12 Перевести LLM generation в `POST /api/chat` на OpenAI-compatible Chat Completions streaming.
- [x] 4.13 Отключить gzip middleware для `/api/chat` и добавить SSE no-buffer/no-transform headers.
- [x] 4.14 Добавить тест gzip passthrough для `/api/chat`.

## 5. Chat Context Assembly

- [x] 5.1 Переписать сборку RAG context на основе ranked hybrid results.
- [x] 5.2 Добавить форматирование fragments с heading/snippet/content для LLM context.
- [x] 5.3 Сохранить общий token budget около 4000 токенов.
- [x] 5.4 Реализовать корректное поведение при пустом/недостаточном контексте.
- [x] 5.5 Добавить unit-тесты context assembly: links, notes, article chunks, duplicate node coverage, empty context.

## 6. Frontend API Client

- [x] 6.1 Добавить TypeScript-типы для index status `keyword_index`.
- [x] 6.2 Добавить `searchKnowledgeBase`/эквивалент для `POST /api/search`.
- [x] 6.3 Расширить `ChatSource` типом, title, fragments.
- [x] 6.4 Расширить `streamChat`, чтобы он мог отправлять `source_paths`.
- [x] 6.5 Добавить unit-тесты/тесты service layer для нового search client и обновлённого chat parsing.

## 7. Search UI

- [x] 7.1 Добавить route `/search` и вкладку «Поиск» в Navbar при доступности индекса.
- [x] 7.2 Создать страницу поиска с input, loading/error/empty states.
- [x] 7.3 Добавить фильтры по type, path/теме и manual_processed.
- [x] 7.4 Отобразить результаты карточками: title, type, annotation, path, keywords, match reasons.
- [x] 7.5 Отобразить article fragments: heading/snippet и источник совпадения.
- [x] 7.6 Добавить переход по title карточки на страницу ноды с сохранением returnTo.
- [x] 7.7 Добавить действие “Спросить по этим источникам” с передачей query/source_paths в чат.
- [x] 7.8 Добавить frontend-тесты SearchPage: success, empty, unavailable, action to chat.
- [x] 7.9 Добавить раскрываемый diagnostics block “Как выполнен поиск”.
- [x] 7.10 Показывать score/rank, match reasons, source kinds и fragment score на карточках.
- [x] 7.11 Сворачивать хвост результатов после заметного перепада score.

## 8. Chat UI

- [x] 8.1 Обновить ChatPage для initial message/source hints из route state.
- [x] 8.2 Обновить отображение sources: title, type, path, fragments.
- [x] 8.3 Добавить раскрытие найденного контекста/fragments.
- [x] 8.4 Показать “недостаточно данных в базе” как нормальное состояние, не техническую ошибку.
- [x] 8.5 Добавить frontend-тесты ChatPage для sources with fragments и source_paths flow.
- [x] 8.6 Переделать ChatPage в message feed: user/assistant bubbles, markdown rendering, stop during streaming.
- [x] 8.7 Перенести sources под конкретный assistant response.
- [x] 8.8 Добавить плашку выбранных источников из поиска и reset-кнопку с tooltip.
- [x] 8.9 Убрать лишнюю прокрутку пустого состояния чата.

## 9. Verification

- [x] 9.1 Запустить `go test ./...`.
- [x] 9.2 Запустить frontend tests (`npm test` или существующую команду проекта).
- [x] 9.3 Запустить frontend build (`npm run build` в `web/`).
- [x] 9.4 Запустить `openspec validate add-hybrid-search-rag-ui`.
- [x] 9.5 Ручная проверка: поиск по точному keyword, semantic-only поиск, переход поиск → чат.
- [x] 9.6 Ручная проверка Search UI diagnostics и score-cliff collapse через Playwright.
- [x] 9.7 Ручная проверка Chat UI: sources under answer, reset source scope, пустой чат без скролла.
- [x] 9.8 Ручная проверка SSE headers `/api/chat`: без gzip, `text/event-stream`, `X-Accel-Buffering: no`.
- [x] 9.9 Ручная проверка rebuild индекса после всех изменений.
