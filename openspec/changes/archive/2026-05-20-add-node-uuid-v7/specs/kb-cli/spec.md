## Purpose

Дельта: CLI для миграции id существующих узлов.

## ADDED Requirements

### Requirement: Подкоманда migrate-node-ids

kb-cli MUST предоставлять подкоманду `migrate-node-ids` для одноразового присвоения поля `id` (UUID v7) всем узлам и файлам переводов в указанной базе (`--path` или `KB_DATA_PATH`), у которых поле `id` отсутствует. Подкоманда MUST поддерживать флаг `--dry-run` (только отчёт без записи). Подкоманда MUST выводить количество обработанных файлов и список конфликтов (дубликаты id или source_url если проверяются).

#### Scenario: Dry-run

- **WHEN** выполняется `kb-cli migrate-node-ids --path /data/kb --dry-run`
- **THEN** файлы не изменяются, в stdout перечислены пути файлов без id

#### Scenario: Применение миграции

- **WHEN** выполняется `kb-cli migrate-node-ids --path /data/kb`
- **THEN** каждый узел без id получает уникальный UUID v7 в frontmatter, updated сохраняется или обновляется по политике Store

#### Scenario: Повторный запуск идемпотентен

- **WHEN** migrate-node-ids выполняется повторно на базе, где у всех узлов уже есть id
- **THEN** ни один файл не изменяется, отчёт показывает 0 изменений
