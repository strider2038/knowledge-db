## Purpose

Стабильный идентификатор узла (UUID v7), уникальность, миграция, дедупликация create/update при ingestion, связь оригинал↔перевод.

## Requirements

## MODIFIED Requirements

### Requirement: Дедупликация ingestion по source_url и id

При сохранении результата ingestion система MUST применять порядок: (1) если передан существующий `node_id` — обновить узел с этим id; (2) иначе если нормализованный `source_url` непустой, результат имеет `type` равный `article` или `link`, и в индексе/хранилище есть узел с этим url — обновить найденный узел, сохранив его `id`; (3) иначе — создать новый узел с новым id. Для `type: note` (и при пустом `type`) система MUST NOT выполнять автоматический lookup и update по `source_url`. При update система MUST NOT создавать второй файл для того же `source_url`.

#### Scenario: Повторный ingest той же статьи по URL

- **WHEN** ingestion обрабатывает материал с `type: article` или `type: link`, для которого уже существует узел с тем же нормализованным `source_url`
- **THEN** обновляется существующий markdown-файл и frontmatter, `id` не меняется, новый файл не создаётся

#### Scenario: Первый ingest URL без существующего узла

- **WHEN** ingestion обрабатывает материал с `type: article` или `type: link`, для которого нет узла с таким `source_url`
- **THEN** создаётся новый узел с новым `id` и записанным `source_url`

#### Scenario: Ingest заметки без URL

- **WHEN** ingestion обрабатывает текст без `source_url`
- **THEN** создаётся новый узел с новым `id` (автодедуп по url не применяется)

#### Scenario: Ingest заметки с source_url, совпадающим с существующим узлом

- **WHEN** ingestion обрабатывает материал с `type: note` и непустым `source_url`, для которого в индексе уже есть узел с тем же нормализованным `source_url`
- **THEN** создаётся новый узел с новым `id`; существующий узел MUST NOT изменяться

#### Scenario: Явный update по node_id

- **WHEN** вызывающий код передаёт известный `node_id` существующего узла
- **THEN** обновляется узел с этим id независимо от `source_url` и `type`
