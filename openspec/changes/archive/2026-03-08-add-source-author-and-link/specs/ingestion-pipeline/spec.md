## Purpose

Pipeline добавления записей в базу: текст или URL → создание узла с файлами. LLM-оркестратор определяет тип контента и действия через function calling.

## Requirements

## MODIFIED Requirements

### Requirement: Интерфейс Ingester

Система ДОЛЖНА (SHALL) предоставлять интерфейс Ingester с методами IngestText(ctx, req) и IngestURL(ctx, url). IngestText MUST принимать IngestRequest с полями Text (обязательно), SourceURL и SourceAuthor (опционально — метаданные источника). Оба метода MUST возвращать созданный узел (*kb.Node) или ошибку.

IngestText — основной метод. Принимает произвольный текст (может содержать URL, инструкции, заметки) и опциональные метаданные источника. LLM-оркестратор внутри pipeline анализирует текст и определяет тип контента и действия.

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

#### Сценарий: Вызов IngestText с URL на сервис/ресурс

- **WHEN** вызывается IngestText с текстом с URL на сервис (не статья)
- **THEN** LLM-оркестратор вызывает fetch_url_meta (title + description), генерирует аннотацию и создаёт узел с type=link

#### Сценарий: Вызов IngestText с инструкциями

- **WHEN** вызывается IngestText с текстом, содержащим URL и инструкции (напр. "сохрани в go/concurrency")
- **THEN** LLM-оркестратор учитывает инструкции при выборе темы и обработке контента

#### Сценарий: Вызов IngestURL

- **WHEN** вызывается IngestURL с URL
- **THEN** система извлекает контент по URL, генерирует метаданные через LLM и создаёт узел в базе; при наличии Author в FetchResult MUST сохранять его как source_author

### Requirement: LLM-оркестратор с function calling

Система ДОЛЖНА (SHALL) использовать LLM (OpenAI-совместимый API, библиотека github.com/openai/openai-go/v3, Responses API) для оркестрации ingestion. LLM MUST получать на вход текст пользователя и контекст базы (существующие темы, существующие keywords) и через function calling определять действия.

Доступные инструменты (tools) для LLM:
- `fetch_url_content(url)` — полное извлечение контента из URL (для статей); LLM получает превью первых 2000 символов для анализа метаданных, включая Author; полный контент кешируется и сохраняется автоматически
- `fetch_url_meta(url)` — извлечение title + description из `<meta>` тегов (для ссылок-закладок)
- `create_node(keywords, annotation, theme_path, slug, type, title, source_url, source_date, source_author, content)` — создание узла; title — обязателен, при отсутствии в источнике (заметка, пересланное сообщение) LLM MUST сгенерировать заголовок на основе контента; annotation — 2–5 предложений; source_author — опционально; для articles поле content ДОЛЖНО быть пустым — полный контент подставляется из кеша fetch_url_content; при слиянии с кешем Author из FetchResult MUST подставляться в source_author, если LLM не вернул значение

#### Сценарий: LLM выбирает fetch_url_content

- **WHEN** LLM определяет, что пользователь хочет сохранить статью по URL
- **THEN** LLM вызывает fetch_url_content, получает превью контента для анализа метаданных; при вызове create_node поле content остаётся пустым, а полный контент подставляется из кеша автоматически; Author из FetchResult подставляется в source_author

#### Сценарий: LLM выбирает fetch_url_meta

- **WHEN** LLM определяет, что URL — ссылка на сервис/ресурс (не статья для копирования)
- **THEN** LLM вызывает fetch_url_meta для получения title и description, формирует аннотацию

#### Сценарий: LLM создаёт заметку без URL

- **WHEN** текст не содержит URL
- **THEN** LLM формирует create_node с type=note, контентом из текста и сгенерированными метаданными

#### Сценарий: LLM использует существующие keywords

- **WHEN** в базе уже есть keyword "goroutines" и новый контент связан с горутинами
- **THEN** LLM MUST использовать существующий keyword "goroutines", а не создавать синоним

#### Сценарий: LLM использует существующие темы

- **WHEN** в базе есть тема "go/concurrency" и контент связан с конкурентностью в Go
- **THEN** LLM MUST предпочитать существующую тему, если она подходит по смыслу

### Requirement: Язык метаданных

LLM ДОЛЖЕН (SHALL) генерировать метаданные на русском языке. Поле `annotation` MUST быть на русском и содержать 2–5 предложений. Поле `keywords` MUST быть на русском; специфичные технические термины, аббревиатуры и имена собственные (TTS, API, Docker и т.п.) допускается оставлять на английском или дублировать на обоих языках. Поле `title` MUST быть заполнено всегда: при наличии в источнике (fetch_url_content, fetch_url_meta) — использовать его; при отсутствии (заметка, пересланное сообщение) — LLM MUST сгенерировать заголовок на основе контента. Язык title — язык оригинального контента.

#### Сценарий: Аннотация на русском

- **WHEN** LLM формирует метаданные для статьи на любом языке
- **THEN** поле `annotation` содержит 2–5 предложений на русском языке

#### Сценарий: Ключевые слова на русском

- **WHEN** LLM формирует keywords для технической статьи
- **THEN** keywords содержат русскоязычные термины; технические аббревиатуры (TTS, API, k8s) допустимы на английском

#### Сценарий: Генерация title при отсутствии в источнике

- **WHEN** контент не содержит явного заголовка (заметка, пересланное сообщение без title)
- **THEN** LLM MUST сгенерировать осмысленный title на основе содержимого и передать его в create_node

### Requirement: Создание узла при ingestion

При успешной обработке система ДОЛЖНА (SHALL) создать узел в базе знаний: директорию с файлом `{slug}.md`, frontmatter с метаданными и markdown-контентом. После создания MUST выполнить git add + commit. Frontmatter MUST содержать поля `title` и `aliases` (если LLM вернул title). Frontmatter MUST содержать `source_author` при наличии в результате create_node (из LLM или подставлено из FetchResult).

#### Сценарий: Создание узла для статьи

- **WHEN** pipeline обработал URL статьи
- **THEN** создаётся узел с type=article, source_url, source_date, source_author (если извлечён), полным контентом и git commit

#### Сценарий: Создание узла для заметки

- **WHEN** pipeline обработал текст без URL
- **THEN** создаётся узел с type=note, контентом из текста и git commit

#### Сценарий: Создание узла для ссылки

- **WHEN** pipeline обработал URL на ресурс/сервис
- **THEN** создаётся узел с type=link, source_url, аннотацией и git commit

#### Сценарий: Создание узла с title

- **WHEN** LLM вернул непустой title в create_node
- **THEN** frontmatter создаваемого узла содержит поля `title` и `aliases: [<title>]`

#### Сценарий: Создание узла с source_author

- **WHEN** create_node вернул source_author (от LLM или подставлен из FetchResult)
- **THEN** frontmatter создаваемого узла содержит поле `source_author`

#### Сценарий: Тема не существует

- **WHEN** LLM указал тему, которой нет в базе
- **THEN** система создаёт промежуточные директории для новой темы
