## Purpose

Добавить набор MCP tools для практической работы кодинговых агентов и чатботов с базой знаний: текстовый/hybrid поиск и семантический поиск.

## Requirements

## ADDED Requirements

### Requirement: MCP tool search_notes

Система ДОЛЖНА (SHALL) предоставлять MCP tool `search_notes` для поиска по заметкам, используя существующий keyword/hybrid retrieval. Tool MUST принимать как минимум `query` и `limit`, и MAY принимать фильтры (`path`, `types`, `manual_processed`) при их поддержке текущим retrieval API.

#### Scenario: Успешный поиск по заметкам

- **WHEN** MCP-клиент вызывает `search_notes` с валидным `query`
- **THEN** сервер возвращает ранжированный список релевантных узлов с полями пути, заголовка, типа и фрагментов

#### Scenario: Пустой query

- **WHEN** MCP-клиент вызывает `search_notes` с пустым `query`
- **THEN** сервер возвращает ошибку валидации инструмента

### Requirement: MCP tool semantic_search

Система ДОЛЖНА (SHALL) предоставлять MCP tool `semantic_search` для семантического поиска по векторному индексу. Tool MUST использовать embeddings/retrieval компоненты проекта и возвращать релевантные результаты в формате MCP tool response.

#### Scenario: Семантический поиск при включённых эмбеддингах

- **WHEN** MCP-клиент вызывает `semantic_search` и в системе доступна embedding-конфигурация
- **THEN** сервер возвращает результаты семантического поиска по базе знаний

#### Scenario: Семантический поиск при отключённых эмбеддингах

- **WHEN** MCP-клиент вызывает `semantic_search`, но `KB_EMBEDDING_ENABLED=false` или embedding provider недоступен
- **THEN** сервер возвращает явную ошибку недоступности semantic-поиска с рекомендацией использовать `search_notes`

### Requirement: MCP tool get_note

Система ДОЛЖНА (SHALL) предоставлять MCP tool `get_note` для чтения содержимого узла базы знаний по `path`. Tool MUST возвращать как минимум `path`, `title`, `content` и MAY возвращать дополнительные метаданные (`type`, `annotation`, `source_url`, `keywords`). Tool MAY принимать `include_content` (по умолчанию `true`) и ограничение длины контента (`max_chars`) с явным признаком усечения.

#### Scenario: Успешное чтение заметки по path

- **WHEN** MCP-клиент вызывает `get_note` с валидным `path`
- **THEN** сервер возвращает содержимое узла и доступные метаданные

#### Scenario: Пустой path

- **WHEN** MCP-клиент вызывает `get_note` с пустым `path`
- **THEN** сервер возвращает ошибку валидации инструмента

#### Scenario: Вызов с include_content=false

- **WHEN** MCP-клиент вызывает `get_note` с `include_content=false`
- **THEN** сервер возвращает метаданные узла без полного текста (`content` пустой, `truncated=false`)

#### Scenario: Узел не найден

- **WHEN** MCP-клиент вызывает `get_note` с path несуществующего узла
- **THEN** сервер возвращает явную ошибку `node not found`
