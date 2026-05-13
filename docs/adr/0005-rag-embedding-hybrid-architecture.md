# ADR 0005: RAG/embedding архитектура (SQLite index + hybrid retrieval)

- Status: accepted
- Date: 2026-05-04
- Supersedes: -
- Superseded-By: -

## Context

Файловый поиск перестал масштабироваться по качеству выдачи для естественных запросов. Нужен был RAG-слой без отказа от markdown как primary storage.

## Decision

Введен опциональный SQLite-индекс для embeddings/chunks/поисковых структур и retrieval-сервис, который эволюционировал к hybrid-search (keyword/FTS + vector + fusion). Индекс остается перестраиваемым вторичным слоем над файловой базой.

## Consequences

### Плюсы

- Существенный рост качества поиска и RAG-контекста.
- Сохранен offline-first принцип для source data.
- Четкое разделение primary storage и search acceleration layer.

### Минусы

- Появились миграции и lifecycle второго хранилища.
- Нужны контроль актуальности индекса и процессы reindex.

## Alternatives

- Внешняя векторная БД: отклонена в пользу embedded/offline-friendly модели.
- Только vector retrieval: отклонен в пользу гибридного, более устойчивого поиска.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-02-add-rag-semantic-search/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-02-add-rag-semantic-search/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-02-add-rag-semantic-search/tasks.md)
- [design.md (hybrid)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-04-add-hybrid-search-rag-ui/design.md)
