## MODIFIED Requirements

### Requirement: Интерфейс Ingester

Система ДОЛЖНА (SHALL) предоставлять интерфейс Ingester с методами IngestText(ctx, text) и IngestURL(ctx, url). Оба метода MUST возвращать созданный узел (*kb.Node) или ошибку.

IngestText — основной метод. Принимает произвольный текст (может содержать URL, инструкции, заметки). LLM-оркестратор внутри pipeline анализирует текст и определяет тип контента и действия.

IngestURL — явный метод для обработки URL. Используется при вызове через API, когда URL известен заранее.

#### Сценарий: Вызов IngestText с текстом-заметкой

- **WHEN** вызывается IngestText с текстом без URL
- **THEN** LLM-оркестратор определяет type=note, генерирует метаданные и создаёт узел в базе

#### Сценарий: Вызов IngestText с URL на статью

- **WHEN** вызывается IngestText с текстом, содержащим URL на статью
- **THEN** LLM-оркестратор вызывает fetch_url_content, извлекает контент статьи, генерирует метаданные и создаёт узел с type=article

#### Сценарий: Вызов IngestText с URL на сервис/ресурс

- **WHEN** вызывается IngestText с URL на сервис (не статья)
- **THEN** LLM-оркестратор вызывает fetch_url_meta (title + description), генерирует аннотацию и создаёт узел с type=link

#### Сценарий: Вызов IngestText с инструкциями

- **WHEN** вызывается IngestText с текстом, содержащим URL и инструкции (напр. "сохрани в go/concurrency")
- **THEN** LLM-оркестратор учитывает инструкции при выборе темы и обработке контента

#### Сценарий: Вызов IngestURL

- **WHEN** вызывается IngestURL с URL
- **THEN** система извлекает контент по URL, генерирует метаданные через LLM и создаёт узел в базе

## REMOVED Requirements

### Requirement: Заглушка в scaffold

**Reason**: Заменена полноценной реализацией pipeline с LLM-оркестрацией.
**Migration**: Использовать PipelineIngester вместо StubIngester. При отсутствии LLM-конфигурации сервер запускается с StubIngester в read-only режиме.

## ADDED Requirements

### Requirement: LLM-оркестратор с function calling

Система ДОЛЖНА (SHALL) использовать LLM (OpenAI-совместимый API, библиотека github.com/openai/openai-go/v3, Responses API) для оркестрации ingestion. LLM MUST получать на вход текст пользователя и контекст базы (существующие темы, существующие keywords) и через function calling определять действия.

Доступные инструменты (tools) для LLM:
- `fetch_url_content(url)` — полное извлечение контента из URL (для статей)
- `fetch_url_meta(url)` — извлечение title + description из `<meta>` тегов (для ссылок-закладок)
- `create_node(keywords, annotation, theme, slug, type, source_url, source_date, content)` — создание узла

#### Сценарий: LLM выбирает fetch_url_content

- **WHEN** LLM определяет, что пользователь хочет сохранить статью по URL
- **THEN** LLM вызывает fetch_url_content, получает контент и формирует параметры для create_node

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

### Requirement: Извлечение контента из URL

Система ДОЛЖНА (SHALL) извлекать контент из URL для оффлайн-хранения. MUST использовать Jina Reader API как основной способ и go-readability + html-to-markdown как fallback.

#### Сценарий: Успешное извлечение через Jina

- **WHEN** ContentFetcher вызывается с URL и Jina Reader API доступен
- **THEN** система возвращает FetchResult с title, markdown-контентом, датой публикации и автором

#### Сценарий: Fallback на go-readability

- **WHEN** Jina Reader API недоступен (таймаут, rate limit, ошибка)
- **THEN** система извлекает контент через go-readability + html-to-markdown и возвращает FetchResult

#### Сценарий: Извлечение мета-данных URL

- **WHEN** вызывается fetch_url_meta для URL
- **THEN** система извлекает title и description из HTML `<meta>` тегов без полного парсинга контента

### Requirement: Создание узла при ingestion

При успешной обработке система ДОЛЖНА (SHALL) создать узел в базе знаний: директорию с файлом `{slug}.md`, frontmatter с метаданными и markdown-контентом. После создания MUST выполнить git add + commit.

#### Сценарий: Создание узла для статьи

- **WHEN** pipeline обработал URL статьи
- **THEN** создаётся узел с type=article, source_url, source_date, полным контентом и git commit

#### Сценарий: Создание узла для заметки

- **WHEN** pipeline обработал текст без URL
- **THEN** создаётся узел с type=note, контентом из текста и git commit

#### Сценарий: Создание узла для ссылки

- **WHEN** pipeline обработал URL на ресурс/сервис
- **THEN** создаётся узел с type=link, source_url, аннотацией и git commit

#### Сценарий: Тема не существует

- **WHEN** LLM указал тему, которой нет в базе
- **THEN** система создаёт промежуточные директории для новой темы

### Requirement: Git sync с remote

Система ДОЛЖНА (SHALL) периодически синхронизировать локальный git-репозиторий базы с remote. Интервал синхронизации MUST быть конфигурируемым через `GIT_SYNC_INTERVAL` (по умолчанию 5 минут).

#### Сценарий: Успешная синхронизация

- **WHEN** наступает интервал синхронизации и remote доступен
- **THEN** система выполняет git fetch + merge

#### Сценарий: Конфликт при merge

- **WHEN** git merge обнаруживает конфликт
- **THEN** система логирует warning и оставляет конфликт для ручного разрешения

#### Сценарий: Remote недоступен

- **WHEN** git fetch не может подключиться к remote
- **THEN** система логирует ошибку и повторяет попытку на следующем интервале; локальные коммиты сохраняются

### Requirement: Конфигурация LLM

Система ДОЛЖНА (SHALL) читать параметры LLM из переменных окружения: `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`.

#### Сценарий: Запуск без LLM-конфигурации

- **WHEN** kb-server запущен без LLM_API_KEY
- **THEN** система запускается с StubIngester, логирует warning; ingestion возвращает ошибку "not implemented"

#### Сценарий: Запуск с LLM-конфигурацией

- **WHEN** kb-server запущен с заданными LLM_API_URL, LLM_API_KEY, LLM_MODEL
- **THEN** система создаёт PipelineIngester с реальным LLM-оркестратором
