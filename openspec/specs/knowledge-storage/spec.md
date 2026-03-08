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

Главный .md файл узла MUST содержать YAML frontmatter с полями: keywords (массив), created (ISO 8601), updated (ISO 8601), type (тип контента: "article", "link", "note"), title (человекочитаемый заголовок), annotation (опционально — краткое описание). Дополнительные опциональные поля: source_url (URL источника), source_date (дата оригинала, ISO 8601), source_author (автор источника: имя, username, название канала и т.п.), aliases (список псевдонимов, совместим с Obsidian). Тело файла — markdown-контент (основное содержание).

- `created` — дата добавления записи в базу знаний
- `source_date` — дата создания оригинального контента (если известна)
- `source_author` — автор источника (автор статьи, канал Telegram, username и т.п.)
- `type` — классификация контента: `article` (копия статьи), `link` (ссылка-закладка), `note` (личная заметка)
- `title` — человекочитаемый заголовок на естественном языке (не slug); используется Obsidian 1.4+
- `aliases` — список псевдонимов узла (Obsidian-совместимо); первый элемент совпадает с `title`; позволяет искать и ссылаться на узел по естественному имени, а не по slug

#### Сценарий: Валидный frontmatter

- **WHEN** frontmatter содержит валидный YAML с обязательными полями keywords, created, updated, type, title
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Отсутствует обязательное поле

- **WHEN** во frontmatter отсутствует поле created, updated, type или title
- **THEN** валидация сообщает об ошибке

#### Сценарий: Frontmatter с опциональными полями

- **WHEN** frontmatter содержит keywords, created, updated, type, title и дополнительно source_url, source_date, source_author, annotation, aliases
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Минимальный валидный frontmatter

- **WHEN** frontmatter содержит только обязательные поля keywords, created, updated, type, title (без source_url, source_date, source_author, aliases)
- **THEN** узел проходит валидацию метаданных

### Requirement: Создание узла через Store

Store ДОЛЖЕН (SHALL) предоставлять метод CreateNode для программного создания узлов. Метод MUST принимать параметры: ThemePath, Slug, Frontmatter, Content. Метод MUST создавать файл `{basePath}/{themePath}/{slug}.md` с frontmatter и markdown-контентом (без slug-директории).

Структура хранения — плоская: узлы хранятся как `{themePath}/{slug}.md` непосредственно в директории темы. Дополнительные файлы (вложения) могут храниться в директории `{themePath}/{slug}/` — она не считается подтемой, если рядом существует `{slug}.md`.

#### Сценарий: Создание узла в существующей теме

- **WHEN** вызывается CreateNode с ThemePath="go/concurrency" и Slug="goroutine-leaks"
- **THEN** создаётся файл `go/concurrency/goroutine-leaks.md` с указанным frontmatter и контентом (без slug-директории)

#### Сценарий: Создание узла в новой теме

- **WHEN** вызывается CreateNode с ThemePath="rust/async" (тема не существует)
- **THEN** создаются промежуточные директории `rust/async/` и файл узла внутри

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
