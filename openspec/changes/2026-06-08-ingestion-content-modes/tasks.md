# Tasks: ingestion-content-modes

## 1. Content mode resolution (backend)

- [ ] 1.1 Добавить `ContentMode` (`verbatim`, `full_fetch`, `digest`, `link_bookmark`, `auto`) в `internal/ingestion` и поле `ContentMode` в `IngestRequest` / `ProcessInput`
- [ ] 1.2 Сохранить в pipeline исходное пользовательское тело отдельно от prompt text (`RawContent` / `OriginalText`), чтобы verbatim guardrail не сохранял служебные префиксы
- [ ] 1.3 Реализовать `ResolveContentMode(input, classification) ContentMode` с детерминированными правилами из `design.md`
- [ ] 1.4 Пробросить `content_mode` из API (`POST /api/ingest`), import accept и Telegram auto-flow; MCP ingest tool не реализовывать в этом change
- [ ] 1.5 Unit-тесты `ResolveContentMode` на матрицу сценариев (paste+article, telegram long-form, URL-only article, URL-only profile, URL-only bookmark, explicit override)

## 2. Mode-specific LLM prompts

- [ ] 2.1 Разделить system/user prompt в `internal/ingestion/llm/prompt.go` по `content_mode` (verbatim / digest / link_bookmark / full_fetch)
- [ ] 2.2 Убрать противоречие «сохранить markdown» vs «переписать в digest» для `verbatim`
- [ ] 2.3 Для `link_bookmark` prompt должен требовать короткое semantic body из доступных фактов, без full-fetch и без пустого `content`
- [ ] 2.4 Обновить `openspec/specs/ingestion-pipeline/spec.md` (через delta) и синхронизировать после merge

## 3. Post-LLM guardrails

- [ ] 3.1 `ensureArticleContent`: не вызывать fetch при `verbatim`; при resolved `full_fetch` fetch/cache является источником body даже если `RawContent` непустой
- [ ] 3.2 Ввести `ensureModeContent`: persisted body непустой для всех modes; для `digest` — structured digest retry, для `link_bookmark` — compact semantic body retry, для `verbatim` — body из `RawContent`
- [ ] 3.3 Вызывать `ensureModeContent` на initial ingest и refresh; refresh mode выводить из stored `type`/`content_profile`/`source_url`/body по таблице `design.md`
- [ ] 3.4 `stripMarkdownFromTitle` / `normalizeTitle`: emoji, markdown links, trailing punctuation — единая функция до persist
- [ ] 3.5 `fetch_url_meta` / GitHub README: не перезаписывать явный `description` из Telegram при `verbatim`/`digest` с телом; выбирать более информативный preview

## 4. API и UI

- [ ] 4.1 Опциональный `content_mode` в `POST /api/ingest` (default `auto`); request поля остаются `text`, `source_url`, `source_author`, `type_hint`
- [ ] 4.2 Invalid non-empty `content_mode` возвращает HTTP 400 `invalid content_mode`; legacy unknown `type_hint` продолжает трактоваться как `auto`
- [ ] 4.3 Ответ `POST /api/ingest` сделать envelope `{ node, content_mode }`; `content_mode` — resolved value, не persisted frontmatter
- [ ] 4.4 Add page: selector режима (Авто / Как есть / Полная статья / Выжимка / Закладка) как primary body-control, `type_hint` как secondary storage hint с подсказками
- [ ] 4.5 Import session accept: принимать `content_mode`, возвращать `{ node, next_item, content_mode }`, добавить selector режима в import tab

## 5. Telegram

- [ ] 5.1 Long-form paste + URL → `verbatim` по умолчанию (без digest rewrite)
- [ ] 5.2 URL-only forward → `full_fetch`, `digest` или `link_bookmark` по resolver; `link_bookmark` создаёт короткое semantic body
- [ ] 5.3 Не добавлять inline-кнопки в обязательный scope этого change; если добавляются позже, они должны передавать тот же `content_mode` enum

## 6. Документация и спеки

- [ ] 6.1 `docs/concepts/ingestion-workflows.md` — концепт четырёх workflow (этот PR)
- [ ] 6.2 ADR `docs/adr/0011-ingestion-content-modes.md` после утверждения design
- [ ] 6.3 Проверить, что `content_mode` не меняет frontmatter contract; `.agents/skills/knowledge-db/SKILL.md` и embedded skill не обновлять в этом change
- [ ] 6.4 `openspec validate 2026-06-08-ingestion-content-modes`

## 7. Регрессии по issues

- [ ] 7.1 Paste + `type=article` + YouTube URL → тело из paste, без scrape (`hermes-desktop-doklad`)
- [ ] 7.2 Telegram long text → verbatim body (`gemma-4-lokalnyj-ii-na-8gb-vram`)
- [ ] 7.3 Title без emoji/markdown (`httptrace-...`)
- [ ] 7.4 Forward с телом → не пустой body (`plagin-bezopasnosti-dlya-claude`)
- [ ] 7.5 URL-only bookmark → непустое compact semantic body, доступное для semantic search
