## Purpose

Определяет формат и структуру хранения базы знаний в файловой системе. База под git, оффлайн-first. Формат совместим с Obsidian: один .md файл на узел с YAML frontmatter.

## Requirements

## ADDED Requirements

### Requirement: Файлы переводов

Система ДОЛЖНА (SHALL) поддерживать файлы переводов в формате `{slug}.{lang}.md` (напр. `{slug}.ru.md`) в той же директории темы, что и оригинальный узел `{slug}.md`. Store MUST предоставлять метод CreateTranslationFile для создания файла перевода. Файл перевода MUST содержать YAML frontmatter с полями: `translation_of` (slug оригинала), `lang` (код языка, напр. "ru"), а также дублированные метаданные из оригинала (keywords, annotation, title, aliases, created, updated, type, source_url, source_date, source_author). В конце тела перевода MUST быть wikilink на оригинал: `[[{slug}|Original]]`.

Оригинальный узел при наличии перевода MUST содержать в frontmatter поле `translations` (массив slug переводов, напр. `[{slug}.ru]`) и в конце тела wikilink на перевод: `[[{slug}.ru|Русский перевод]]`.

#### Сценарий: Создание файла перевода

- **WHEN** вызывается CreateTranslationFile с themePath, slug, lang="ru", frontmatter и content
- **THEN** создаётся файл `{themePath}/{slug}.ru.md` с указанным frontmatter и контентом

#### Сценарий: Frontmatter перевода

- **WHEN** создаётся файл перевода
- **THEN** frontmatter содержит обязательные поля `translation_of`, `lang` и дублированные поля из оригинала

#### Сценарий: Связь оригинала и перевода

- **WHEN** создан перевод для узла
- **THEN** в оригинальном файле добавлены поле `translations` и wikilink на перевод; в файле перевода — wikilink на оригинал
