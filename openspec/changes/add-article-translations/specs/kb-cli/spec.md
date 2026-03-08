## Purpose

Консольная утилита kb-cli для валидации структуры базы и инициализации новой базы (gitignore, установка skills).

## Requirements

## ADDED Requirements

### Requirement: Валидация файлов переводов

kb-cli validate ДОЛЖЕН (SHALL) проверять файлы переводов (`*.ru.md`). Файл считается переводом, если в той же директории существует `{slug}.md`, где `{slug}` — часть имени до `.ru.md`. Для каждого файла перевода система MUST проверять: frontmatter содержит обязательные поля `translation_of` и `lang`; `translation_of` совпадает со slug оригинала; все wikilinks в теле (`[[target]]`, `[[target|label]]`) ведут на существующие узлы базы. Если у узла есть файл перевода, в frontmatter оригинала MUST быть поле `translations`, содержащее slug перевода. Нарушения выводятся в общий отчёт validate.

#### Сценарий: Валидный файл перевода

- **WHEN** файл `slug.ru.md` имеет `translation_of: slug`, `lang: ru`, оригинал `slug.md` существует, wikilinks ведут на существующие узлы, в оригинале есть `translations: [slug.ru]`
- **THEN** валидация проходит без ошибок

#### Сценарий: Отсутствует translation_of

- **WHEN** в frontmatter перевода отсутствует поле `translation_of`
- **THEN** validate сообщает об ошибке

#### Сценарий: Wikilink на несуществующий узел

- **WHEN** в теле перевода есть `[[missing-node]]` и узла `missing-node` нет в базе
- **THEN** validate сообщает об ошибке

#### Сценарий: Оригинал без translations

- **WHEN** существует `slug.ru.md`, но в frontmatter `slug.md` отсутствует поле `translations`
- **THEN** validate сообщает об ошибке
