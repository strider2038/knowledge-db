# ADR 0007: Chat sessions/memory как временное SQLite-хранилище

- Status: accepted
- Date: 2026-05-12
- Supersedes: -
- Superseded-By: -

## Context

Чат требовал UX persistent-сессий, но при этом не должен был становиться частью постоянного git-knowledge слоя.

## Decision

Введена отдельная доменная подсистема `chat-session-memory` с хранением в SQLite и ограничением контекстного окна (trim/summarize). Chat-memory считается временным рабочим контекстом, а не долговременным knowledge storage.

## Consequences

### Плюсы

- Предсказуемый lifecycle (create/list/rename/delete/cleanup).
- Контроль размера контекста и затрат на inference.
- Отделение пользовательских знаний (markdown/git) от диалогового runtime-состояния.

### Минусы

- История чатов не участвует в git-аудите.
- Нужна отдельная политика очистки/TTL.

## Alternatives

- Полная история без сворачивания: отклонена из-за роста контекста и шума.
- Хранить чаты в data/markdown: отклонено для временного вспомогательного сценария.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-chat-memory-and-history/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-chat-memory-and-history/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-chat-memory-and-history/tasks.md)
