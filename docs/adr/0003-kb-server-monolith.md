# ADR 0003: Монолит `kb-server` (API + UI + Telegram + MCP)

- Status: accepted
- Date: 2026-03-06
- Supersedes: -
- Superseded-By: -

## Context

На старте проекта требовалась простая эксплуатация и единая точка запуска локального сервера.

## Decision

Выбрана монолитная модель: `kb-server` объединяет REST API, встроенную web-статику, Telegram-бота и MCP endpoint в одном процессе. Вспомогательная CLI-функциональность вынесена в `kb-cli`.

## Consequences

### Плюсы

- Минимальный операционный overhead для локального запуска.
- Простой lifecycle и конфигурация компонентов.
- Легко держать согласованность API/UI/ingestion.

### Минусы

- Более плотная связность подсистем в одном бинарнике.
- Риски влияния проблем одной подсистемы на весь процесс.

## Alternatives

- Микросервисы по компонентам: отклонены как избыточные для текущего масштаба.
- Отдельное развертывание UI от API: отклонено в пользу embedded сценария.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/tasks.md)
