## MODIFIED Requirements

### Requirement: Формат frontmatter

Главный .md файл узла MUST содержать YAML frontmatter с полями: keywords (массив), created (ISO 8601), updated (ISO 8601), annotation (опционально — краткое описание). Дополнительные опциональные поля: type (тип контента: "article", "link", "note"), source_url (URL источника), source_date (дата оригинала, ISO 8601), title (человекочитаемый заголовок), aliases (список псевдонимов, совместим с Obsidian). Тело файла — markdown-контент (основное содержание).

- `created` — дата добавления записи в базу знаний
- `source_date` — дата создания оригинального контента (если известна)
- `type` — классификация контента: `article` (копия статьи), `link` (ссылка-закладка), `note` (личная заметка)
- `title` — человекочитаемый заголовок на естественном языке (не slug); используется Obsidian 1.4+
- `aliases` — список псевдонимов узла (Obsidian-совместимо); первый элемент совпадает с `title`; позволяет искать и ссылаться на узел по естественному имени, а не по slug

#### Сценарий: Валидный frontmatter

- **WHEN** frontmatter содержит валидный YAML с полями keywords, created, updated
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Отсутствует обязательное поле

- **WHEN** во frontmatter отсутствует поле created или updated
- **THEN** валидация сообщает об ошибке

#### Сценарий: Frontmatter с опциональными полями

- **WHEN** frontmatter содержит keywords, created, updated и дополнительно type, source_url, source_date, title, aliases
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Frontmatter без опциональных полей

- **WHEN** frontmatter содержит только keywords, created, updated (без type, source_url, source_date, title, aliases)
- **THEN** узел проходит валидацию (обратная совместимость)

## ADDED Requirements

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
