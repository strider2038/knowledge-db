## Purpose

Определяет формат и структуру хранения базы знаний в файловой системе. База под git, оффлайн-first.

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

Каждый узел (папка со статьёй/заметкой) MUST содержать: `annotation.md`, `content.md`, `metadata.json`. Дополнительно допускаются: подпапка `notes/` с `.md` файлами, подпапка `images/`, подпапка `.local/` (исключена из git).

#### Сценарий: Валидный узел

- **WHEN** узел содержит annotation.md, content.md, metadata.json
- **THEN** узел считается валидным

#### Сценарий: Отсутствует обязательный файл

- **WHEN** в узле отсутствует content.md
- **THEN** валидация сообщает об ошибке

### Requirement: Исключение .local из git

Директория `.local/` в каждом узле MUST быть исключена из git (через .gitignore в корне базы). В ней хранятся sha-хеш, embedding и прочие вспомогательные файлы.

#### Сценарий: .gitignore в корне базы

- **WHEN** в корне базы есть .gitignore с правилами `**/.local/`, `**/.local/**`
- **THEN** содержимое .local не попадает в репозиторий

### Requirement: Формат metadata.json

Файл metadata.json MUST содержать поля: source (опционально), sourceType, keywords (массив), created, updated (ISO 8601).

#### Сценарий: Валидный metadata.json

- **WHEN** metadata.json содержит валидный JSON с полями keywords, created, updated
- **THEN** узел проходит валидацию метаданных
