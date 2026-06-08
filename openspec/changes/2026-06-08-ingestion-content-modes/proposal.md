## Why

Ingestion pipeline смешивает четыре разных пользовательских намерения в одном LLM-оркестраторе: verbatim-сохранение текста, полную копию статьи с URL, digest/выжимку и профиль внешнего ресурса. Это приводит к регрессиям: вставленный транскрипт затирается fetch по `source_url`, длинные Telegram-посты переписываются в `conceptual_digest` без явного запроса, digest-note сохраняются с пустым телом при первичном ingest, заголовки из каналов остаются с emoji/markdown.

Открытые debug-issue подтверждают системный характер проблемы, а не единичные сбои модели.

## What Changes

- Ввести явную ось **content mode** (`verbatim`, `full_fetch`, `digest`, `link_bookmark`) — отдельно от `type` (`article|link|note`) и `content_profile`.
- Определять content mode **детерминированно** в pipeline до вызова LLM (эвристики + `TypeHint` + `content_mode` + текстовые инструкции пользователя).
- Исправить `ensureArticleContent`: не подменять уже предоставленное пользователем тело при `full_fetch`, если вход содержит существенный текст.
- Расширить `ensureDigestContent` на первичный ingest и на `type=note` с digest-профилями, не только refresh link-узлов.
- Добавить детерминированную нормализацию `title`/`aliases` после оркестрации (markdown + leading emoji).
- Уточнить промпт оркестратора: убрать противоречие «note без изменений» vs «conceptual_digest переписывает».
- Расширить API/UI ingest опциональным `content_mode` (или эквивалентным явным выбором режима).
- Добавить концепт-документацию workflow в `docs/concepts/`.

## Capabilities

### New Capabilities

- `ingestion-content-modes`: правила выбора режима обработки тела узла, guardrails после LLM, контракт API/UI.

### Modified Capabilities

- `ingestion-pipeline`: классификация, post-LLM guardrails, промпт, ingest/refresh симметрия.
- `rest-api`: опциональное поле `content_mode` в `POST /api/ingest`.
- `webapp`: явный выбор режима на странице добавления (вместо/рядом с неоднозначным `type_hint=article` для paste-сценария).

## Impact

- `internal/ingestion/profile.go`, `pipeline.go`, `llm/prompt.go`, `llm/orchestrator.go`
- `internal/api/handlers.go` (ingest request)
- `web/src/pages/AddPage.tsx`, `web/src/services/api.ts`
- `docs/concepts/ingestion-workflows.md`
- OpenSpec delta: `ingestion-pipeline`, `rest-api`, `webapp`
- Совместимость: `content_mode=auto` по умолчанию; существующие узлы не мигрируются автоматически
