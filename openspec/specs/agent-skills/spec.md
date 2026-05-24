## Purpose

Agent skill для Cursor/Claude — локальная работа с базой знаний из IDE. Skill содержит инструкции по формату и структуре узлов.

## Requirements

### Requirement: Формат SKILL.md

Skill MUST быть в формате SKILL.md с frontmatter и описанием назначения.

#### Сценарий: Чтение skill агентом

- **WHEN** агент (Cursor/Claude) загружает skill
- **THEN** он получает инструкции по работе с базой

### Requirement: Путь к базе

Skill ДОЛЖЕН (SHALL) содержать путь к базе знаний, подставляемый при установке (шаблон {{DATA_PATH}}).

#### Сценарий: Установка с подстановкой пути

- **WHEN** `kb init` копирует skill
- **THEN** {{DATA_PATH}} заменяется на фактический путь к базе

### Requirement: Инструкции по созданию узлов

Skill MUST описывать структуру узла согласно `knowledge-storage`: плоский файл `{theme}/{slug}.md` с YAML frontmatter (обязательные поля `id`, `keywords`, `created`, `updated`, `type`, `title`), опциональная директория `{slug}/` для вложений (`images/`, `notes/`), `.local/` в gitignore.

#### Сценарий: Агент создаёт узел

- **WHEN** агент создаёт новый узел по инструкциям skill
- **THEN** создаётся `{theme}/{slug}.md` с валидным frontmatter и телом; при необходимости — вложения в `{slug}/`

### Requirement: Расположение skill в базе знаний

Канонический шаблон skill хранится в репозитории приложения (`.agents/skills/knowledge-db/SKILL.md`, embedded в `kb`). `kb init` MUST копировать только skill `knowledge-db` в `{KB}/.agents/skills/knowledge-db/SKILL.md`, не в `~/.cursor/skills/`.

#### Сценарий: Init устанавливает skill в базу

- **WHEN** `kb init --path /path/to/kb`
- **THEN** файл `{kb}/.agents/skills/knowledge-db/SKILL.md` существует и не содержит плейсхолдер `{{DATA_PATH}}`
