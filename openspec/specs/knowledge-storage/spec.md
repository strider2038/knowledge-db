## Purpose

Определяет формат и структуру хранения базы знаний в файловой системе. База под git, оффлайн-first. Формат совместим с Obsidian: один .md файл на узел с YAML frontmatter.
## Requirements
### Requirement: Иерархия тем

Система ДОЛЖНА (SHALL) хранить знания в иерархии тем: директории тем, внутри — подтемы. Глубина вложенности MUST быть не более 2–3 уровней.

#### Сценарий: Валидная структура тем

- **WHEN** база содержит topic/subtopic/node
- **THEN** структура считается валидной

#### Сценарий: Слишком глубокая вложенность

- **WHEN** база содержит topic/subtopic/subsubtopic/subsubsubtopic
- **THEN** валидация сообщает об ошибке

### Requirement: Структура узла

Структура хранения — плоская: каждый узел MUST храниться как файл `{slug}.md` непосредственно в директории темы (без slug-поддиректории). Файл MUST содержать YAML frontmatter и markdown-тело. Дополнительно рядом с файлом узла MAY существовать директория `{slug}/` для вложений (images, notes) — она не считается подтемой. Директория `.local/` внутри вложений исключена из git.

#### Сценарий: Валидный узел

- **WHEN** в директории темы существует файл `{slug}.md` с валидным frontmatter (keywords, created, updated)
- **THEN** узел считается валидным

#### Сценарий: Отсутствует главный файл

- **WHEN** для записи в теме не найден файл `{slug}.md`
- **THEN** валидация сообщает об ошибке

#### Сценарий: Невалидный frontmatter

- **WHEN** главный .md файл не содержит обязательных полей (keywords, created, updated) во frontmatter
- **THEN** валидация сообщает об ошибке

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

### Requirement: Исключение .local из git

Директория `.local/` в каждом узле MUST быть исключена из git (через .gitignore в корне базы). В ней хранятся sha-хеш, embedding и прочие вспомогательные файлы.

#### Сценарий: .gitignore в корне базы

- **WHEN** в корне базы есть .gitignore с правилами `**/.local/`, `**/.local/**`
- **THEN** содержимое .local не попадает в репозиторий

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

### Requirement: Профиль внешнего источника во frontmatter

Главный markdown-файл узла MAY содержать опциональные frontmatter-поля `source_kind` и `content_profile`. Поле `source_kind` MUST описывать природу внешнего источника и при наличии иметь одно из значений: `repository`, `documentation`, `product_service`, `online_tool`, `directory_catalog`, `learning_resource`, `article`, `news`, `social_post`, `unknown`. Поле `content_profile` MUST описывать локальную форму digest и при наличии иметь одно из значений: `repository_profile`, `product_profile`, `documentation_profile`, `online_tool_profile`, `directory_profile`, `learning_resource_profile`, `conceptual_digest`, `brief_digest`, `link_bookmark`.

#### Scenario: Узел с repository profile

- **WHEN** frontmatter содержит `type: link`, `source_kind: repository`, `content_profile: repository_profile`
- **THEN** узел проходит валидацию метаданных

#### Scenario: Концептуальная заметка по статье

- **WHEN** frontmatter содержит `type: note`, `source_kind: article`, `content_profile: conceptual_digest`
- **THEN** узел проходит валидацию метаданных

#### Scenario: Старый узел без профиля

- **WHEN** frontmatter содержит обязательные поля, но не содержит `source_kind` и `content_profile`
- **THEN** узел остаётся валидным

#### Scenario: Невалидное значение source_kind

- **WHEN** поле `source_kind` присутствует и содержит значение вне допустимого списка
- **THEN** валидация метаданных сообщает об ошибке

#### Scenario: Невалидное значение content_profile

- **WHEN** поле `content_profile` присутствует и содержит значение вне допустимого списка
- **THEN** валидация метаданных сообщает об ошибке

### Requirement: Markdown-тело для link digest

Узел `type=link` MAY содержать markdown-тело с профильным digest внешнего ресурса. Если `content_profile` присутствует и не равен `link_bookmark`, тело SHOULD содержать человекочитаемое концептуальное описание ресурса. Пустое тело для обычной закладки MUST оставаться допустимым.

#### Scenario: Link с профильным телом

- **WHEN** узел `type=link` содержит `content_profile: repository_profile` и markdown-тело
- **THEN** узел проходит валидацию и тело сохраняется как часть знания

#### Scenario: Обычная закладка без тела

- **WHEN** узел `type=link` не содержит `content_profile` и имеет пустое тело
- **THEN** узел остаётся валидным

### Requirement: Нормализация labels при записи

Store MUST нормализовать `labels` перед сохранением в frontmatter: trim каждого элемента; удаление пустых строк; дедупликация без учёта регистра с сохранением первого варианта написания; максимум 32 метки на узел; длина каждой метки после trim не более 64 символов. Символ запятой в метке MUST запрещаться. Пустой массив после нормализации MUST приводить к удалению ключа `labels` из frontmatter.

#### Сценарий: Дедупликация без учёта регистра

- **WHEN** пользователь сохраняет labels `["Favorite", "favorite", "review"]`
- **THEN** в frontmatter сохраняется `labels: ["Favorite", "review"]`

#### Сценарий: Очистка меток

- **WHEN** после нормализации не остаётся ни одной метки
- **THEN** ключ `labels` отсутствует в frontmatter

#### Сценарий: Превышение лимита

- **WHEN** после нормализации меток больше 32 или длина метки превышает 64 символа
- **THEN** операция записи возвращает ошибку валидации

### Requirement: labels не участвуют в семантическом индексе контента

Изменение только поля `labels` MUST NOT изменять content_hash узла для embedding-индекса. Поле `labels` MUST NOT включаться в текст для генерации embedding и MUST NOT включаться в searchable text для keyword/FTS/RAG по смыслу контента.

#### Сценарий: Смена метки без переиндексации embedding

- **WHEN** у узла изменены только `labels`, без изменения title, annotation, keywords, type, body
- **THEN** content_hash для embedding остаётся прежним и переиндексация embedding не требуется

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

