## Why

Ingestion pipeline смешивает четыре разных пользовательских намерения в одном LLM-оркестраторе: verbatim-сохранение текста, полную копию статьи с URL, digest/выжимку и профиль внешнего ресурса. Это приводит к регрессиям: вставленный транскрипт затирается fetch по `source_url`, длинные Telegram-посты переписываются в `conceptual_digest` без явного запроса, digest-note сохраняются с пустым телом при первичном ingest, заголовки из каналов остаются с emoji/markdown.

Открытые debug-issue подтверждают системный характер проблемы, а не единичные сбои модели.

## What Changes

- Ввести явную ось **content mode** (`auto`, `verbatim`, `full_fetch`, `digest`, `link_bookmark`) — отдельно от `type` (`article|link|note`) и `content_profile`.
- Определять content mode **детерминированно** в pipeline до вызова LLM (эвристики + `TypeHint` + `content_mode` + текстовые инструкции пользователя).
- Исправить `ensureArticleContent`: не подменять уже предоставленное пользователем тело при resolved `verbatim`; explicit `content_mode=full_fetch` остаётся override и может заменить body fetch-результатом.
- Заменить узкий digest-guardrail на `ensureModeContent`: persisted body всегда непустой; для `digest` это структурированный digest, для `link_bookmark` — компактное semantic body.
- Добавить детерминированную нормализацию `title`/`aliases` после оркестрации (markdown + leading emoji).
- Уточнить промпт оркестратора: убрать противоречие «note без изменений» vs «conceptual_digest переписывает».
- Расширить API/UI/import ingest опциональным `content_mode` (или эквивалентным явным выбором режима).
- Возвращать resolved `content_mode` в API/import responses и logs; не сохранять `content_mode` во frontmatter.
- Добавить концепт-документацию workflow в `docs/concepts/`.

## Capabilities

### New Capabilities

- `ingestion-content-modes`: правила выбора режима обработки тела узла, guardrails после LLM, контракт API/UI/import.

### Modified Capabilities

- `ingestion-pipeline`: классификация, post-LLM guardrails, промпт, ingest/refresh симметрия.
- `ingest-type-hint`: `type_hint` остаётся storage-form hint и больше не определяет обработку тела.
- `rest-api`: опциональное поле `content_mode` в `POST /api/ingest` и import accept.
- `webapp`: явный выбор режима на странице добавления и в import session (вместо/рядом с неоднозначным `type_hint=article` для paste-сценария).
- `telegram-bot`: auto-resolver для live-сообщений; inline-кнопки выбора режима вне обязательного scope.

## Impact

- `internal/ingestion/profile.go`, `pipeline.go`, `llm/prompt.go`, `llm/orchestrator.go`
- `internal/api/handlers.go`, `internal/api/import_handlers.go` (ingest/import request and response envelope)
- `internal/import/session/store.go`, `internal/telegram/bot.go`
- `web/src/pages/AddPage.tsx`, `web/src/services/api.ts`
- `docs/concepts/ingestion-workflows.md`
- OpenSpec delta: `ingestion-content-modes`, `ingestion-pipeline`, `ingest-type-hint`, `rest-api`, `webapp`, `telegram-bot`
- Совместимость: `content_mode=auto` по умолчанию; существующие узлы не мигрируются автоматически; `content_mode` не пишется во frontmatter
- Out of scope: MCP ingest tool, потому что текущий MCP server не предоставляет ingest tool
