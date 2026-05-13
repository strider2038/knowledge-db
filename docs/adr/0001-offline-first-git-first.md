# ADR 0001: Offline-first + Git-first как базовый принцип

- Status: accepted
- Date: 2026-03-06
- Supersedes: -
- Superseded-By: -

## Context

Проект создавался как персональная база знаний, которая должна оставаться работоспособной локально и полностью контролироваться пользователем.

## Decision

База знаний хранится в файловой структуре markdown/frontmatter под git. Git является источником правды для версионирования, diff и merge. Сетевые зависимости допускаются только для опциональных сценариев ingest/LLM, но не для чтения и управления локальными знаниями.

## Consequences

### Плюсы

- Данные доступны локально и читаемы без сервера.
- История изменений прозрачна и совместима со стандартными git-процессами.
- Минимальная привязка к внешней инфраструктуре.

### Минусы

- Синхронизация и конфликты решаются на уровне git-практик.
- Некоторые онлайн-функции (ingest URL, LLM) не работают офлайн по определению.

## Alternatives

- Централизованная БД как primary storage: отклонена, т.к. снижает portability и контроль пользователя.
- Cloud-first синхронизация как обязательный слой: отклонена, т.к. противоречит локальному базовому сценарию.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/tasks.md)
