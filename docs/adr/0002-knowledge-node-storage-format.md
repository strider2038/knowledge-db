# ADR 0002: Формат хранения knowledge nodes (Obsidian-compatible)

- Status: accepted
- Date: 2026-03-06
- Supersedes: -
- Superseded-By: -

## Context

Нужна была совместимость с Obsidian без конвертеров и без потери структуры узлов.

## Decision

Узел хранится как директория с главным markdown-файлом `{dirname}.md`, содержащим YAML frontmatter и body. Дополнительные материалы (например, `notes/`, `images/`) остаются рядом в директории узла. Файловая система и markdown остаются source of truth.

## Consequences

### Плюсы

- Нативная читаемость в Obsidian и других markdown-инструментах.
- Единая модель хранения для сервера, CLI и ручного редактирования.
- Хорошая merge-совместимость с git.

### Минусы

- Требуется строгая валидация frontmatter.
- Структура директорий глубже, чем при flat-файлах.

## Alternatives

- Старый тройной формат (`annotation.md` + `content.md` + `metadata.json`): отклонен как менее совместимый с Obsidian.
- `index.md` как имя главного файла: отклонен в пользу более читаемого `{dirname}.md`.

## References

- [design.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-obsidian-compatible-storage/design.md)
- [proposal.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-obsidian-compatible-storage/proposal.md)
- [tasks.md](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-obsidian-compatible-storage/tasks.md)
