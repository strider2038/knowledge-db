## Why

Текущий RAG-слой умеет искать по векторной близости и отвечать через чат, но не смешивает semantic retrieval с точными совпадениями по ключевикам, заголовкам и терминам. Из-за этого база знаний может пропускать важные заметки с редкими словами, названиями библиотек, аббревиатурами и точными формулировками, а UI пока не даёт удобного режима “найти и изучить материалы” отдельно от чата.

## What Changes

- Добавить гибридный retrieval pipeline: keyword/FTS candidates + vector candidates + fusion/ranking.
- Добавить API для гибридного поиска, который возвращает карточки нод и релевантные фрагменты без генерации LLM-ответа.
- Перевести RAG-чат на общий retrieval pipeline, чтобы ответы использовали тот же набор источников, что и поиск.
- Добавить relevance cutoff/порог уверенности, чтобы чат не отвечал на нерелевантном контексте.
- Добавить LLM query rewrite для гибридного поиска с локальными vocabulary hints из базы, чтобы пользовательские вопросы лучше совпадали с терминами индекса.
- Добавить отдельный UI-режим “Поиск” с карточками статей/ссылок/заметок, фильтрами и переходом к вопросу по найденным источникам.
- Улучшить UI поиска: показывать diagnostics/rewrite/meta, score/reasons/source kinds на карточках и сворачивать хвост выдачи после заметного перепада score.
- Улучшить UI чата: форматировать интерфейс как обычный чат, отображать источники под ответом, позволять сбрасывать ограничение выбранных источников и раскрывать найденные фрагменты.
- Улучшить streaming чата для OpenAI-compatible локальных провайдеров (например LM Studio): использовать chat completions streaming и отключать gzip/buffering для SSE.
- Сохранить offline-first/git-first: markdown остаётся источником правды, индекс остаётся опциональным и перестраиваемым.

## Capabilities

### New Capabilities

- `hybrid-search`: гибридный поиск по базе знаний, объединяющий точные совпадения, полнотекстовые/ключевые совпадения и semantic vector search.

### Modified Capabilities

- `embedding-index`: индекс должен хранить данные, необходимые для keyword/FTS поиска и выдачи фрагментов, а не только embeddings/chunks.
- `rag-chat`: чат должен использовать общий гибридный retrieval pipeline, пороги релевантности и более прозрачные источники.
- `rest-api`: API должен предоставить endpoint гибридного поиска и расширить контракт RAG endpoints/status для UI.
- `webapp`: веб-интерфейс должен получить отдельный режим гибридного поиска и улучшенный чатовый UX.

## Impact

- Backend: `internal/index`, `internal/api`, `internal/bootstrap`, тесты API и index-layer.
- Frontend: `web/src/services/api.ts`, `web/src/pages`, `web/src/components`, маршрутизация и Navbar.
- SQLite index: возможное расширение схемы под searchable text/FTS, snippets, source metadata и unified ranked results.
- RAG: контекстная сборка должна опираться на гибридные результаты, а не на два независимых vector top-K.
- LLM provider compatibility: chat generation должен учитывать OpenAI-compatible провайдеров, которые лучше поддерживают `/v1/chat/completions`, чем Responses API.
- API tests обязательны для новых/изменённых endpoint'ов.
