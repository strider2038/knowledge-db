## Purpose

Консольная утилита kb-cli для валидации структуры базы и инициализации новой базы (gitignore, установка skills).

## Requirements

### Requirement: Подкоманда validate

kb-cli ДОЛЖЕН (SHALL) предоставлять подкоманду validate для проверки структуры базы.

#### Сценарий: Валидация указанного пути

- **WHEN** kb-cli validate --path /path/to/base
- **THEN** проверяется структура и выводится отчёт об ошибках или успехе

#### Сценарий: Валидация текущей директории

- **WHEN** kb-cli validate (без --path) в директории базы
- **THEN** проверяется текущая директория

### Requirement: Подкоманда init

kb-cli ДОЛЖЕН (SHALL) предоставлять подкоманду init для инициализации новой базы знаний.

#### Сценарий: Init в указанной директории

- **WHEN** kb-cli init --path /path/to/base
- **THEN** создаётся .gitignore с правилами `**/.local/`, `**/.local/**`, копируются agent skills в ~/.cursor/skills/ с подстановкой пути

#### Сценарий: Init в текущей директории

- **WHEN** kb-cli init (без --path)
- **THEN** инициализация выполняется в текущей директории

### Requirement: Идемпотентность init

init MUST быть идемпотентным: повторный вызов не должен ломать существующую конфигурацию.

#### Сценарий: Повторный init

- **WHEN** kb-cli init вызывается повторно в той же директории
- **THEN** .gitignore и skills обновляются, без потери данных

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

### Requirement: Подкоманда dump-images

kb-cli ДОЛЖЕН (SHALL) предоставлять подкоманду dump-images для скачивания удалённых изображений из markdown-статьи и замены ссылок на локальные пути. Подробная спецификация — в `kb-cli-dump-images`.

#### Сценарий: Наличие подкоманды

- **WHEN** пользователь вызывает kb-cli dump-images --help
- **THEN** отображается справка по флагам --path, --file, --dry-run
