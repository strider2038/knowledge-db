# ADR 0011: Content modes для ingestion (verbatim / fetch / digest / bookmark)

- Status: proposed
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
3. Post-LLM guardrails по таблице mode (не перезаписывать paste, digest на ingest, и т.д.).
4. Детерминированная нормализация `title`/`aliases` после оркестрации.
5. Опциональное поле `content_mode` в `POST /api/ingest` и селектор на Add page.

`type`, `source_kind`, `content_profile` (ADR 0010) сохраняются; content_mode описывает источник тела и допустимость переписывания.

## Consequences

### Плюсы

- Предсказуемое поведение для paste, Telegram и URL-only сценариев.
- Тестируемая матрица без полной зависимости от LLM.
- Симметрия ingest и refresh для digest.

### Минусы

- Дополнительная ось и эвристики resolver; пограничные кейсы требуют явного override.
- UI и API surface расширяются.

## Alternatives

- Только улучшить промпт без mode: отклонено — guardrails в коде всё равно перезаписывают тело.
- Новый `type` вместо mode: отклонено — ломает совместимость и не отделяет «форма хранения» от «источник тела».

## References

- [proposal.md](../../openspec/changes/2026-06-08-ingestion-content-modes/proposal.md)
- [design.md](../../openspec/changes/2026-06-08-ingestion-content-modes/design.md)
- [tasks.md](../../openspec/changes/2026-06-08-ingestion-content-modes/tasks.md)
- [ingestion-workflows.md](../concepts/ingestion-workflows.md)
