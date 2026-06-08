# ADR 0011: Content modes для ingestion (verbatim / fetch / digest / bookmark)

- Status: accepted
- Date: 2026-06-08
- Supersedes: -
- Superseded-By: -

## Context

После ADR 0010 ingestion научился строить digest для link и note, но ось «как обрабатывать тело узла» осталась неявной. LLM-промпт смешивает четыре пользовательских намерения; post-LLM guardrails применяются несимметрично (`ensureDigestContent` только на refresh link, `ensureArticleContent` затирает paste при `type=article`).

Концепт: [ingestion-workflows.md](../concepts/ingestion-workflows.md).

## Decision

Ввести явную ось **content_mode** (`verbatim`, `full_fetch`, `digest`, `link_bookmark`, `auto`):

1. **ResolveContentMode** в pipeline до LLM — детерминированные правила + override из API/UI.
2. Промпт и tool-flow зависят от mode; убрать конфликт verbatim vs digest rewrite.
3. Post-LLM guardrails по таблице mode: не перезаписывать paste, digest на ingest/refresh, compact body для bookmark.
4. Детерминированная нормализация `title`/`aliases` после оркестрации.
5. Опциональное поле `content_mode` в `POST /api/ingest` и import accept; selector режима на Add page.

`type`, `source_kind`, `content_profile` (ADR 0010) сохраняются; content_mode описывает источник тела и допустимость переписывания.

Persisted `content` узла всегда должен быть непустым, чтобы semantic search и RAG могли находить узел. Поэтому `link_bookmark` означает не пустую закладку, а короткое semantic body из доступных фактов.

`content_mode` не сохраняется во frontmatter в этом решении. Он существует в request, resolved response и logs/job logs. Refresh-description выводит mode из stored `type`, `content_profile` и `source_url`; body emptiness является repair trigger после выбора mode.

## Consequences

### Плюсы

- Предсказуемое поведение для paste, Telegram и URL-only сценариев.
- Тестируемая матрица без полной зависимости от LLM.
- Симметрия ingest и refresh для digest/bookmark body guardrails.
- Узлы остаются пригодными для semantic search, потому что пустой body не сохраняется.

### Минусы

- Дополнительная ось и эвристики resolver; пограничные кейсы требуют явного override.
- UI и API surface расширяются.
- API response `POST /api/ingest` меняется на envelope `{ node, content_mode }`, поэтому web client должен обновиться вместе с backend.

## Alternatives

- Только улучшить промпт без mode: отклонено — guardrails в коде всё равно перезаписывают тело.
- Новый `type` вместо mode: отклонено — ломает совместимость и не отделяет «форма хранения» от «источник тела».

## References

- [proposal.md](../../openspec/changes/2026-06-08-ingestion-content-modes/proposal.md)
- [design.md](../../openspec/changes/2026-06-08-ingestion-content-modes/design.md)
- [tasks.md](../../openspec/changes/2026-06-08-ingestion-content-modes/tasks.md)
- [ingestion-workflows.md](../concepts/ingestion-workflows.md)
