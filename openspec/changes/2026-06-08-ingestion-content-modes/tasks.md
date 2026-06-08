# Tasks: ingestion-content-modes

## 1. Content mode resolution (backend)

- [ ] 1.1 Добавить `ContentMode` (`verbatim`, `full_fetch`, `digest`, `link_bookmark`, `auto`) в `internal/ingestion` и поле `ContentMode` в `IngestRequest` / `IngestInput`
- [ ] 1.2 Реализовать `ResolveContentMode(input, classification) ContentMode` с детерминированными правилами из `design.md`
- [ ] 1.3 Пробросить `content_mode` из API (`POST /api/ingest`), Telegram (команды/кнопки), MCP (`ingest` tool) с валидацией enum
- [ ] 1.4 Unit-тесты `ResolveContentMode` на матрицу сценариев (paste+article, telegram long-form, link only, explicit override)

## 2. Mode-specific LLM prompts

- [ ] 2.1 Разделить system/user prompt в `internal/ingestion/llm/prompt.go` по `content_mode` (verbatim / digest / link_bookmark / full_fetch)
- [ ] 2.2 Убрать противоречие «сохранить markdown» vs «переписать в digest» для `verbatim`
- [ ] 2.3 Обновить `openspec/specs/ingestion-pipeline/spec.md` (через delta) и синхронизировать после merge

## 3. Post-LLM guardrails

- [ ] 3.1 `ensureArticleContent`: не вызывать fetch при `verbatim` или при непустом `RawContent` + `type=article`
- [ ] 3.2 `ensureDigestContent`: вызывать на ingest для `digest` и `link_bookmark` (не только refresh); для `verbatim` — пропуск
- [ ] 3.3 `stripMarkdownFromTitle` / `normalizeTitle`: emoji, markdown links, trailing punctuation — единая функция до persist
- [ ] 3.4 `fetch_url_meta` / GitHub README: не перезаписывать явный `description` из Telegram при `verbatim`/`digest` с телом

## 4. API и UI

- [ ] 4.1 Опциональный `content_mode` в `POST /api/ingest` (default `auto`); отразить в OpenAPI/handlers
- [ ] 4.2 Add page: селектор режима (Авто / Дословно / Полная статья / Дайджест / Ссылка) с подсказками
- [ ] 4.3 Отображать resolved `content_mode` в ответе ingest / preview (debug-friendly)

## 5. Telegram

- [ ] 5.1 Long-form paste + URL → `verbatim` по умолчанию (без digest rewrite)
- [ ] 5.2 URL-only forward → `link_bookmark` + digest generation
- [ ] 5.3 Опционально: inline-кнопки «Сохранить как есть» / «Сделать дайджест» после классификации

## 6. Документация и спеки

- [ ] 6.1 `docs/concepts/ingestion-workflows.md` — концепт четырёх workflow (этот PR)
- [ ] 6.2 ADR `docs/adr/0011-ingestion-content-modes.md` после утверждения design
- [ ] 6.3 Обновить `.agents/skills/knowledge-db/SKILL.md` если меняется контракт полей узла
- [ ] 6.4 `openspec validate 2026-06-08-ingestion-content-modes`

## 7. Регрессии по issues

- [ ] 7.1 Paste + `type=article` + YouTube URL → тело из paste, без scrape (`hermes-desktop-doklad`)
- [ ] 7.2 Telegram long text → verbatim body (`gemma-4-lokalnyj-ii-na-8gb-vram`)
- [ ] 7.3 Title без emoji/markdown (`httptrace-...`)
- [ ] 7.4 Forward с телом → не пустой body (`plagin-bezopasnosti-dlya-claude`)
