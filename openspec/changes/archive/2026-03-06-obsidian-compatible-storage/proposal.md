## Why

База знаний сейчас использует формат узла с тремя отдельными файлами (annotation.md, content.md, metadata.json), который несовместим с Obsidian. Obsidian ожидает один .md файл на заметку с YAML frontmatter. Нужна прямая совместимость: чтобы vault можно было открыть в Obsidian без конвертера, и при этом сохранить структуру тем и подтем.

## What Changes

- **BREAKING**: Формат узла меняется: вместо annotation.md + content.md + metadata.json — один главный .md файл с frontmatter
- Узел остаётся папкой: node-name/node-name.md внутри topic/subtopic/
- Метаданные (source, sourceType, keywords, created, updated) переносятся в YAML frontmatter
- annotation и content объединяются в один .md: annotation в frontmatter, content в теле
- notes/, images/, .local/ сохраняются без изменений
- Иерархия тем (2–3 уровня) не меняется

## Capabilities

### New Capabilities

Нет новых capabilities.

### Modified Capabilities

- `knowledge-storage`: изменение структуры узла (Requirement: Структура узла, Requirement: Формат metadata.json)

## Impact

- `internal/kb/`: validate.go, tree.go — логика IsNodeDir, GetNode, валидация
- `internal/api/`: handlers, если они напрямую читают annotation.md/content.md
- `cmd/kb-cli/`: init — шаблоны узлов
- `.cursor/skills/knowledge-db/SKILL.md`: инструкции по структуре узла
- Существующие базы: потребуется миграция или поддержка обоих форматов (обратная совместимость)
