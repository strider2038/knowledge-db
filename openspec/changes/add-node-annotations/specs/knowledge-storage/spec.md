## Purpose

Дельта: sidecar `annotations.yaml` как часть структуры узла.

## MODIFIED Requirements

### Requirement: Структура узла

Структура хранения — плоская: каждый узел MUST храниться как файл `{slug}.md` непосредственно в директории темы (без slug-поддиректории). Файл MUST содержать YAML frontmatter и markdown-тело. Дополнительно рядом с файлом узла MAY существовать директория `{slug}/` для вложений (images, notes) — она не считается подтемой. В директории `{slug}/` MAY существовать файл `annotations.yaml` с личными аннотациями пользователя (git-tracked, отдельно от frontmatter `annotation` и от подпапки `notes/`). Директория `.local/` внутри вложений исключена из git.

#### Сценарий: Валидный узел

- **WHEN** в директории темы существует файл `{slug}.md` с валидным frontmatter (keywords, created, updated)
- **THEN** узел считается валидным

#### Сценарий: Отсутствует главный файл

- **WHEN** для записи в теме не найден файл `{slug}.md`
- **THEN** валидация сообщает об ошибке

#### Сценарий: Невалидный frontmatter

- **WHEN** главный .md файл не содержит обязательных полей (keywords, created, updated) во frontmatter
- **THEN** валидация сообщает об ошибке

#### Сценарий: Узел с sidecar аннотаций

- **WHEN** существует `topic/article.md` и `topic/article/annotations.yaml` с `version: 1`
- **THEN** узел считается валидным; sidecar не участвует в валидации frontmatter главного файла
