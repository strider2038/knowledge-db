## Purpose

Дельта: личные метки узлов (`labels`) в frontmatter — отдельно от семантических `keywords`.

## MODIFIED Requirements

### Requirement: Формат frontmatter

Главный .md файл узла MUST содержать YAML frontmatter с полями: keywords (массив), created (ISO 8601), updated (ISO 8601), type (тип контента: "article", "link", "note"), title (человекочитаемый заголовок), annotation (опционально — краткое описание). Дополнительные опциональные поля: source_url (URL источника), source_date (дата оригинала, ISO 8601), source_author (автор источника: имя, username, название канала и т.п.), aliases (список псевдонимов, совместим с Obsidian), manual_processed (boolean — пользователь отметил запись как проверенную вручную с точки зрения сортировки и обработки), labels (массив строк — личные метки узла, не участвуют в семантическом поиске и RAG). Тело файла — markdown-контент (основное содержание).

- `created` — дата добавления записи в базу знаний
- `source_date` — дата создания оригинального контента (если известна)
- `source_author` — автор источника (автор статьи, канал Telegram, username и т.п.)
- `type` — классификация контента: `article` (копия статьи), `link` (ссылка-закладка), `note` (личная заметка)
- `title` — человекочитаемый заголовок на естественном языке (не slug); используется Obsidian 1.4+
- `aliases` — список псевдонимов узла (Obsidian-совместимо); первый элемент совпадает с `title`; позволяет искать и ссылаться на узел по естественному имени, а не по slug
- `manual_processed` — если `true`, узел считается отмеченным пользователем как прошедший ручную проверку; если поле отсутствует или `false` — не отмечен
- `labels` — произвольные пользовательские метки (избранное, «перечитать» и т.д.); только на узлах, не на темах; не смешиваются с `keywords`

#### Сценарий: Валидный frontmatter

- **WHEN** frontmatter содержит валидный YAML с обязательными полями keywords, created, updated, type, title
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Отсутствует обязательное поле

- **WHEN** во frontmatter отсутствует поле created, updated, type или title
- **THEN** валидация сообщает об ошибке

#### Сценарий: Frontmatter с опциональными полями

- **WHEN** frontmatter содержит keywords, created, updated, type, title и дополнительно source_url, source_date, source_author, annotation, aliases, manual_processed, labels
- **THEN** узел проходит валидацию метаданных

#### Сценарий: Минимальный валидный frontmatter

- **WHEN** frontmatter содержит только обязательные поля keywords, created, updated, type, title (без source_url, source_date, source_author, aliases, manual_processed, labels)
- **THEN** узел проходит валидацию метаданных

#### Сценарий: manual_processed только true или false

- **WHEN** во frontmatter указано manual_processed со значением, отличным от boolean true/false
- **THEN** валидация сообщает об ошибке

## ADDED Requirements

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
