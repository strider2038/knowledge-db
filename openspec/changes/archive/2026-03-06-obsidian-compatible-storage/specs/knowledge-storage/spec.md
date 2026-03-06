## Purpose

Определяет формат и структуру хранения базы знаний в файловой системе. База под git, оффлайн-first. Формат совместим с Obsidian: один .md файл на узел с YAML frontmatter.

## Requirements

## MODIFIED Requirements

### Requirement: Структура узла

Каждый узел (папка со статьёй/заметкой) MUST содержать главный файл `{dirname}.md`, где dirname — имя папки узла. Файл содержит YAML frontmatter и markdown-тело. Дополнительно допускаются: подпапка `notes/` с `.md` файлами, подпапка `images/`, подпапка `.local/` (исключена из git).

#### Сценарий: Валидный узел

- **WHEN** узел содержит файл `{dirname}.md` с валидным frontmatter (keywords, created, updated)
- **THEN** узел считается валидным

#### Сценарий: Отсутствует главный файл

- **WHEN** в узле отсутствует `{dirname}.md`
- **THEN** валидация сообщает об ошибке

#### Сценарий: Невалидный frontmatter

- **WHEN** главный .md файл не содержит обязательных полей (keywords, created, updated) во frontmatter
- **THEN** валидация сообщает об ошибке

### Requirement: Формат frontmatter

Главный .md файл узла MUST содержать YAML frontmatter с полями: source (опционально), sourceType, keywords (массив), created, updated (ISO 8601), annotation (опционально — краткое описание). Тело файла — markdown-контент (основное содержание).

#### Сценарий: Валидный frontmatter

- **WHEN** frontmatter содержит валидный YAML с полями keywords, created, updated
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Отсутствует обязательное поле

- **WHEN** во frontmatter отсутствует поле created или updated
- **THEN** валидация сообщает об ошибке

## REMOVED Requirements

### Requirement: Формат metadata.json

**Reason:** Метаданные перенесены в YAML frontmatter главного .md файла для совместимости с Obsidian.

**Migration:** Использовать frontmatter в `{dirname}.md` вместо отдельного metadata.json.
