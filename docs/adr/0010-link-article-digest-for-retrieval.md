# ADR 0010: Эволюция link/article digest для retrieval и RAG

- Status: accepted
- Date: 2026-05-12
- Supersedes: -
- Superseded-By: -

## Context

Короткие карточки `type=link` без содержательного body давали слабый контекст для RAG и ухудшали качество retrieval по внешним источникам.

## Decision

Ingestion эволюционирован к profile/digest-подходу:
- для репозиториев/сервисов/документации `type=link` сохраняется, но с профильным digest body;
- для длинных источников без полного копирования применяется концептуальный digest как `type=note`;
- digest body индексируется в keyword/vector/chunk слоях и участвует в RAG.

## Consequences

### Плюсы

- Более плотный и полезный retrieval-контекст.
- Лучше покрываются концептуальные запросы по внешним ресурсам.
- Сохраняется git-friendly markdown модель без переноса полного мусорного контента.

### Минусы

- Выше требования к quality guardrails промптов.
- Риск ошибочной классификации `link` vs `note` при сложных источниках.

## Alternatives

- Хранить только расширенную annotation без body: отклонено, т.к. это слабый RAG-контекст.
- Массовый автоматический backfill как единственный путь: отклонен в пользу контролируемого точечного обновления.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-add-link-profile-digests/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-add-link-profile-digests/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-add-link-profile-digests/tasks.md)
