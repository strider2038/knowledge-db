# node-identity Specification

## Purpose
TBD - created by archiving change add-node-uuid-v7. Update Purpose after archive.
## Requirements
### Requirement: UUID v7 в frontmatter узла

Каждый главный файл узла (`{slug}.md`) и каждый файл перевода (`{slug}.{lang}.md`) MUST содержать во frontmatter поле `id` — строка UUID версии 7 в каноническом lowercase формате (8-4-4-4-12 hex). Система MUST генерировать новый id при программном создании узла, если id не передан явно. Поле `id` MUST NOT изменяться при обновлении контента, перемещении узла (смене path) или переименовании slug.

#### Scenario: Новый узел получает id

- **WHEN** Store создаёт узел через CreateNode без поля id во frontmatter
- **THEN** в записанном frontmatter присутствует валидный UUID v7

#### Scenario: Move сохраняет id

- **WHEN** узел перемещён из `old/topic/slug` в `new/topic/slug`
- **THEN** поле `id` в frontmatter остаётся тем же значением

### Requirement: Уникальность id в базе

Система MUST обеспечивать глобальную уникальность `id` среди всех узлов и файлов переводов в `KB_DATA_PATH`. Валидатор и команда миграции MUST сообщать о конфликтах (два файла с одним id).

#### Scenario: Дубликат id при валидации

- **WHEN** два разных `.md` файла содержат одинаковое поле `id`
- **THEN** validate сообщает об ошибке с путями обоих файлов

### Requirement: Дедупликация ingestion по source_url и id

При сохранении результата ingestion система MUST применять порядок: (1) если передан существующий `node_id` — обновить узел с этим id; (2) иначе если результат имеет `type: link`, нормализованный `source_url` непустой и в индексе/хранилище есть узел с этим url — обновить найденный узел, сохранив его `id`; (3) иначе — создать новый узел с новым id. Для `type: article` и `type: note` (и при пустом `type`) система MUST NOT выполнять автоматический lookup и update по `source_url`. При update система MUST NOT создавать второй файл для того же `source_url`.

#### Scenario: Повторный ingest той же ссылки по URL

- **WHEN** ingestion обрабатывает материал с `type: link`, для которого уже существует узел с тем же нормализованным `source_url`
- **THEN** обновляется существующий markdown-файл и frontmatter, `id` не меняется, новый файл не создаётся

#### Scenario: Первый ingest ссылки без существующего узла

- **WHEN** ingestion обрабатывает материал с `type: link`, для которого нет узла с таким `source_url`
- **THEN** создаётся новый узел с новым `id` и записанным `source_url`

#### Scenario: Ingest статьи с source_url, совпадающим с существующим узлом

- **WHEN** ingestion обрабатывает материал с `type: article` и непустым `source_url`, для которого в индексе уже есть узел с тем же нормализованным `source_url`
- **THEN** создаётся новый узел с новым `id`; существующий узел MUST NOT изменяться

#### Scenario: Ingest заметки без URL

- **WHEN** ingestion обрабатывает текст без `source_url`
- **THEN** создаётся новый узел с новым `id` (автодедуп по url не применяется)

#### Scenario: Ingest заметки с source_url, совпадающим с существующим узлом

- **WHEN** ingestion обрабатывает материал с `type: note` и непустым `source_url`, для которого в индексе уже есть узел с тем же нормализованным `source_url`
- **THEN** создаётся новый узел с новым `id`; существующий узел MUST NOT изменяться

#### Scenario: Явный update по node_id

- **WHEN** вызывающий код передаёт известный `node_id` существующего узла
- **THEN** обновляется узел с этим id независимо от `source_url` и `type`

### Requirement: Связь перевода через translation_of_id

Файл перевода MUST иметь собственный уникальный `id`. Файл перевода MAY содержать поле `translation_of_id` — UUID v7 оригинального узла. Поле `translation_of` (slug) MUST сохраняться для Obsidian и wikilinks. Оригинал и перевод MUST NOT разделять одно значение `id`.

#### Scenario: Перевод с отдельным id

- **WHEN** создаётся файл `{slug}.ru.md` для оригинала с `id` оригинала
- **THEN** frontmatter перевода содержит новый `id` и `translation_of_id`, равный id оригинала

### Requirement: Миграция существующих узлов

Система MUST предоставлять одноразовую команду CLI для присвоения `id` всем узлам без поля `id`. Команда MUST поддерживать dry-run. После применения миграции все узлы MUST иметь `id`.

#### Scenario: Dry-run миграции

- **WHEN** выполняется `kb migrate-node-ids --dry-run`
- **THEN** выводится список файлов, которым будет присвоен id, без изменения файлов

#### Scenario: Применение миграции

- **WHEN** выполняется `kb migrate-node-ids` без dry-run
- **THEN** каждый узел без `id` получает уникальный UUID v7 в frontmatter

