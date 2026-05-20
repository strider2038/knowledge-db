## Purpose

Дельта: идентификатор `id` в frontmatter, генерация при CreateNode, связь переводов.

## ADDED Requirements

### Requirement: Идентификатор узла id

Главный .md файл узла и файл перевода MUST содержать обязательное поле `id` (UUID v7) во frontmatter после внедрения change и выполнения миграции данных. Store MUST возвращать `id` в модели Node и NodeListItem. Store MUST предоставлять метод получения узла по `id` (GetNodeByID).

#### Scenario: Чтение узла с id

- **WHEN** GetNode загружает валидный узел
- **THEN** Metadata содержит поле `id` и API-нормализованный ответ включает `id`

#### Scenario: GetNodeByID

- **WHEN** вызывается GetNodeByID с существующим id
- **THEN** возвращается узел с актуальным path

#### Scenario: GetNodeByID не найден

- **WHEN** id не существует в базе
- **THEN** возвращается ErrNodeNotFound

## MODIFIED Requirements

### Requirement: Формат frontmatter

Главный .md файл узла MUST содержать YAML frontmatter с полями: **id** (UUID v7, обязательное после миграции), keywords (массив), created (ISO 8601), updated (ISO 8601), type (тип контента: "article", "link", "note"), title (человекочитаемый заголовок), annotation (опционально — краткое описание). Дополнительные опциональные поля: source_url (URL источника), source_date (дата оригинала, ISO 8601), source_author (автор источника: имя, username, название канала и т.п.), aliases (список псевдонимов, совместим с Obsidian), manual_processed (boolean — пользователь отметил запись как проверенную вручную с точки зрения сортировки и обработки), labels (массив строк — личные метки узла, не участвуют в семантическом поиске и RAG). Тело файла — markdown-контент (основное содержание).

- `id` — стабильный машинный идентификатор узла; не меняется при move/rename
- `created` — дата добавления записи в базу знаний
- `source_date` — дата создания оригинального контента (если известна)
- `source_author` — автор источника (автор статьи, канал Telegram, username и т.п.)
- `type` — классификация контента: `article` (копия статьи), `link` (ссылка-закладка), `note` (личная заметка)
- `title` — человекочитаемый заголовок на естественном языке (не slug); используется Obsidian 1.4+
- `aliases` — список псевдонимов узла (Obsidian-совместимо); первый элемент совпадает с `title`; позволяет искать и ссылаться на узел по естественному имени, а не по slug
- `manual_processed` — если `true`, узел считается отмеченным пользователем как прошедший ручную проверку; если поле отсутствует или `false` — не отмечен
- `labels` — произвольные пользовательские метки (избранное, «перечитать» и т.д.); только на узлах, не на темах; не смешиваются с `keywords`

#### Сценарий: Валидный frontmatter

- **WHEN** frontmatter содержит валидный YAML с обязательными полями id, keywords, created, updated, type, title
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Отсутствует обязательное поле

- **WHEN** во frontmatter отсутствует поле id, created, updated, type или title
- **THEN** валидация сообщает об ошибке

#### Сценарий: Frontmatter с опциональными полями

- **WHEN** frontmatter содержит id, keywords, created, updated, type, title и дополнительно source_url, source_date, source_author, annotation, aliases, manual_processed, labels
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Минимальный валидный frontmatter

- **WHEN** frontmatter содержит обязательные поля id, keywords, created, updated, type, title (без source_url, source_date, source_author, aliases, manual_processed, labels)
- **THEN** узел проходит валидацию метаданных

#### Сценарий: manual_processed только true или false

- **WHEN** поле manual_processed присутствует и имеет значение не boolean (например, строка или число)
- **THEN** валидация метаданных сообщает об ошибке

#### Сценарий: Невалидный id

- **WHEN** поле id присутствует, но не является валидным UUID
- **THEN** валидация метаданных сообщает об ошибке

### Requirement: Создание узла через Store

Store ДОЛЖЕН (SHALL) предоставлять метод CreateNode для программного создания узлов. Метод MUST принимать параметры: ThemePath, Slug, Frontmatter, Content. Метод MUST создавать файл `{basePath}/{themePath}/{slug}.md` с frontmatter и markdown-контентом (без slug-директории). Если frontmatter не содержит `id`, Store MUST сгенерировать UUID v7 и записать его в frontmatter перед сохранением файла.

Структура хранения — плоская: узлы хранятся как `{themePath}/{slug}.md` непосредственно в директории темы. Дополнительные файлы (вложения) могут храниться в директории `{themePath}/{slug}/` — она не считается подтемой, если рядом существует `{slug}.md`.

#### Сценарий: Создание узла в существующей теме

- **WHEN** вызывается CreateNode с ThemePath="go/concurrency" и Slug="goroutine-leaks"
- **THEN** создаётся файл `go/concurrency/goroutine-leaks.md` с указанным frontmatter, контентом и полем `id`

#### Сценарий: Создание узла в новой теме

- **WHEN** вызывается CreateNode с ThemePath="rust/async" (тема не существует)
- **THEN** создаются промежуточные директории `rust/async/` и файл узла внутри с полем `id`

#### Сценарий: Slug уже существует

- **WHEN** вызывается CreateNode с Slug, который уже существует в указанной теме (существует `{slug}.md`)
- **THEN** система добавляет числовой суффикс (-2, -3 и т.д.) для уникальности

#### Сценарий: Директория вложений рядом с узлом

- **WHEN** в теме существует `long-slug.md` и директория `long-slug/` с файлами
- **THEN** `long-slug/` не считается подтемой и не нарушает валидацию структуры

### Requirement: Файлы переводов

Система ДОЛЖНА (SHALL) поддерживать файлы переводов в формате `{slug}.{lang}.md` (напр. `{slug}.ru.md`) в той же директории темы, что и оригинальный узел `{slug}.md`. Store MUST предоставлять метод CreateTranslationFile для создания файла перевода. Файл перевода MUST содержать YAML frontmatter с полями: **id** (UUID v7, уникальный для файла перевода), `translation_of` (slug оригинала), **translation_of_id** (UUID v7 оригинала, опционально но MUST заполняться при программном создании), `lang` (код языка, напр. "ru"), а также дублированные метаданные из оригинала (keywords, annotation, title, aliases, created, updated, type, source_url, source_date, source_author). В конце тела перевода MUST быть wikilink на оригинал: `[[{slug}|Original]]`.

Оригинальный узел при наличии перевода MUST содержать в frontmatter поле `translations` (массив slug переводов, напр. `[{slug}.ru]`) и в конце тела wikilink на перевод: `[[{slug}.ru|Русский перевод]]`.

#### Сценарий: Создание файла перевода

- **WHEN** вызывается CreateTranslationFile с themePath, slug, lang="ru", frontmatter и content
- **THEN** создаётся файл `{themePath}/{slug}.ru.md` с указанным frontmatter, контентом и уникальным `id`

#### Сценарий: Frontmatter перевода

- **WHEN** создаётся файл перевода программно
- **THEN** frontmatter содержит обязательные поля `id`, `translation_of`, `lang`, `translation_of_id` и дублированные поля из оригинала

#### Сценарий: Связь оригинала и перевода

- **WHEN** создан перевод для узла
- **THEN** в оригинальном файле добавлены поле `translations` и wikilink на перевод; в файле перевода — wikilink на оригинал и `translation_of_id` равен `id` оригинала
