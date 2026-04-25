## Purpose

Дельта к спецификации хранения: опциональный флаг ручной проверки обработки узла в frontmatter.

## Requirements

## MODIFIED Requirements

### Requirement: Формат frontmatter

Главный .md файл узла MUST содержать YAML frontmatter с полями: keywords (массив), created (ISO 8601), updated (ISO 8601), type (тип контента: "article", "link", "note"), title (человекочитаемый заголовок), annotation (опционально — краткое описание). Дополнительные опциональные поля: source_url (URL источника), source_date (дата оригинала, ISO 8601), source_author (автор источника: имя, username, название канала и т.п.), aliases (список псевдонимов, совместим с Obsidian), manual_processed (boolean — пользователь отметил запись как проверенную вручную с точки зрения сортировки и обработки). Тело файла — markdown-контент (основное содержание).

- `created` — дата добавления записи в базу знаний
- `source_date` — дата создания оригинального контента (если известна)
- `source_author` — автор источника (автор статьи, канал Telegram, username и т.п.)
- `type` — классификация контента: `article` (копия статьи), `link` (ссылка-закладка), `note` (личная заметка)
- `title` — человекочитаемый заголовок на естественном языке (не slug); используется Obsidian 1.4+
- `aliases` — список псевдонимов узла (Obsidian-совместимо); первый элемент совпадает с `title`; позволяет искать и ссылаться на узел по естественному имени, а не по slug
- `manual_processed` — если `true`, узел считается отмеченным пользователем как прошедший ручную проверку; если поле отсутствует или `false` — не отмечен

#### Сценарий: Валидный frontmatter

- **WHEN** frontmatter содержит валидный YAML с обязательными полями keywords, created, updated, type, title
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Отсутствует обязательное поле

- **WHEN** во frontmatter отсутствует поле created, updated, type или title
- **THEN** валидация сообщает об ошибке

#### Сценарий: Frontmatter с опциональными полями

- **WHEN** frontmatter содержит keywords, created, updated, type, title и дополнительно source_url, source_date, source_author, annotation, aliases, manual_processed
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Минимальный валидный frontmatter

- **WHEN** frontmatter содержит только обязательные поля keywords, created, updated, type, title (без source_url, source_date, source_author, aliases, manual_processed)
- **THEN** узел проходит валидацию метаданных

#### Сценарий: manual_processed только true или false

- **WHEN** поле manual_processed присутствует и имеет значение не boolean (например, строка или число)
- **THEN** валидация метаданных сообщает об ошибке
