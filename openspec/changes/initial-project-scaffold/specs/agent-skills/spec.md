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

- **WHEN** kb-cli init копирует skill
- **THEN** {{DATA_PATH}} заменяется на фактический путь к базе

### Requirement: Инструкции по созданию узлов

Skill MUST описывать структуру узла (annotation.md, content.md, metadata.json, notes/, images/, .local/) и правила создания.

#### Сценарий: Агент создаёт узел

- **WHEN** агент создаёт новый узел по инструкциям skill
- **THEN** создаётся папка с обязательными файлами в правильном формате
