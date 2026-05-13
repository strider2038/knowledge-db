# ADR 0006: Разделение SQLite: `index.db` и `chat.db`

- Status: accepted
- Date: 2026-05-12
- Supersedes: -
- Superseded-By: -

## Context

После внедрения RAG-индекса и persistent chat sessions в проекте появились два разных класса данных: поисковый индекс и временная память чатов.

## Decision

Использовать раздельные SQLite-файлы в `.kb/`:
- `index.db` для embeddings/retrieval индекса;
- `chat.db` для chat sessions/messages/summary.

Разделение закрепляется как текущее операционное решение.

## Consequences

### Плюсы

- Разные lifecycle: reindex/очистка индекса не затрагивает чат-историю.
- Разные write-паттерны и миграции изолированы.
- Упрощение отказоустойчивости: повреждение одного файла не ломает второй контур.

### Минусы

- Два файла вместо одного для backup/операций.
- Две линии миграций и операционных проверок.

### Operational Notes

Пересмотр решения уместен при выполнении одного или нескольких триггеров:
- устойчивый рост lock contention или I/O overhead от раздельных операций;
- непропорциональная сложность сопровождения двух миграционных потоков;
- явный продуктовый запрос на single-file режим для упрощения эксплуатации/backup.

Если триггеры подтверждены, допустим режим объединения в один SQLite-файл как отдельный ADR/change.

## Alternatives

- Один общий SQLite-файл для индекса и чатов: отложен; может быть полезен для single-file эксплуатации, но повышает связанность lifecycle.
- Хранение chat-memory в файлах data/: отклонено как избыточное для временной подсистемы.

## References

- [design.md (RAG index)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-02-add-rag-semantic-search/design.md)
- [proposal.md (RAG index)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-02-add-rag-semantic-search/proposal.md)
- [design.md (chat memory)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-chat-memory-and-history/design.md)
- [proposal.md (chat memory)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-05-12-chat-memory-and-history/proposal.md)
- [bootstrap.go](/home/strider/projects/knowledge-db/internal/bootstrap/bootstrap.go)
