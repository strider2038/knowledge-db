# ADR 0004: Ingestion pipeline с LLM orchestration и fetch-chain

- Status: accepted
- Date: 2026-03-07
- Supersedes: -
- Superseded-By: -

## Context

Ввод пользователя (особенно Telegram/UI) неструктурирован: текст, URL, комментарии, type hints и смешанные сценарии требовали более гибкого маршрутизации, чем простой regex pipeline.

## Decision

Ingestion реализован как pipeline с LLM-orchestrator и tool-calling, где модель выбирает инструменты (`fetch_url_content`, `fetch_url_meta`, `create_node`) в зависимости от намерения. Для извлечения контента выбрана цепочка fetcher'ов (Jina primary + fallback).

## Consequences

### Плюсы

- Устойчивость к смешанному вводу и пользовательским инструкциям.
- Расширяемость tool-схем без ломки контракта API.
- Более качественное извлечение источников и метаданных.

### Минусы

- Выше сложность отладки и тестирования оркестрации.
- Зависимость качества результата от prompt/tool schema.

## Alternatives

- Жесткий split `IngestText`/`IngestURL` с regex routing: отклонен как недостаточно точный для реальных сценариев.
- Только один fetcher без fallback: отклонен из-за деградации при ошибках внешнего источника.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-07-implement-ingestion-pipeline/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-07-implement-ingestion-pipeline/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-07-implement-ingestion-pipeline/tasks.md)
- [design.md (type hint)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-10-ingest-via-ui/design.md)
