## Purpose

Pipeline добавления записей в базу: текст или URL → создание узла с файлами. LLM-оркестратор определяет тип контента и действия через function calling.

## Requirements

## MODIFIED Requirements

### Requirement: Интерфейс Ingester

Система ДОЛЖНА (SHALL) предоставлять интерфейс Ingester с методами IngestText(ctx, req) и IngestURL(ctx, url). IngestText MUST принимать IngestRequest с полями Text (обязательно), SourceURL, SourceAuthor, TypeHint (опционально — метаданные источника и подсказка типа). Оба метода MUST возвращать созданный узел (*kb.Node) или ошибку.

IngestText — основной метод. Принимает произвольный текст (может содержать URL, инструкции, заметки) и опциональные метаданные источника. Поле TypeHint (опционально): "auto", "article", "link", "note". При TypeHint != "" и != "auto" LLM-оркестратор MUST использовать указанный тип при вызове create_node. LLM-оркестратор внутри pipeline анализирует текст и определяет тип контента и действия.

IngestURL — явный метод для обработки URL. Используется при вызове через API, когда URL известен заранее. При успешном fetch система MUST передавать Author из FetchResult как source_author.

#### Сценарий: Вызов IngestText с текстом-заметкой

- **WHEN** вызывается IngestText с текстом без URL
- **THEN** LLM-оркестратор определяет type=note, генерирует метаданные и создаёт узел в базе

#### Сценарий: Вызов IngestText с URL на статью

- **WHEN** вызывается IngestText с текстом, содержащим URL на статью
- **THEN** LLM-оркестратор вызывает fetch_url_content, извлекает контент статьи, генерирует метаданные и создаёт узел с type=article

#### Сценарий: Вызов IngestText с метаданными источника

- **WHEN** вызывается IngestText с IngestRequest, содержащим SourceURL и/или SourceAuthor (напр. при пересылке из Telegram)
- **THEN** pipeline передаёт метаданные в LLM-контекст и создаёт узел с source_url, source_author в frontmatter при наличии

#### Сценарий: Вызов IngestText с TypeHint

- **WHEN** вызывается IngestText с IngestRequest, содержащим TypeHint = "article", "link" или "note"
- **THEN** оркестратор передаёт подсказку в LLM и создаёт узел с указанным типом

#### Сценарий: Вызов IngestText с URL на сервис/ресурс

- **WHEN** вызывается IngestText с текстом с URL на сервис (не статья)
- **THEN** LLM-оркестратор вызывает fetch_url_meta (title + description), генерирует аннотацию и создаёт узел с type=link

#### Сценарий: Вызов IngestText с инструкциями

- **WHEN** вызывается IngestText с текстом, содержащим URL и инструкции (напр. "сохрани в go/concurrency")
- **THEN** LLM-оркестратор учитывает инструкции при выборе темы и обработке контента

#### Сценарий: Вызов IngestURL

- **WHEN** вызывается IngestURL с URL
- **THEN** система извлекает контент по URL, генерирует метаданные через LLM и создаёт узел в базе; при наличии Author в FetchResult MUST сохранять его как source_author
