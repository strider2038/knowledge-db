## 1. Зависимости

- [x] 1.1 Добавить `github.com/adrg/frontmatter` (или аналог) для парсинга YAML frontmatter в go.mod

## 2. internal/kb

- [x] 2.1 Изменить `IsNodeDir`: проверять наличие `{dirname}.md` вместо annotation.md, content.md, metadata.json
- [x] 2.2 Добавить функцию парсинга frontmatter (keywords, created, updated) и валидации полей
- [x] 2.3 Изменить `GetNode`: читать `{dirname}.md`, парсить frontmatter → Metadata + Annotation, тело → Content
- [x] 2.4 Обновить `Validate` и `validateNode`: валидировать главный .md и frontmatter вместо трёх файлов
- [x] 2.5 Обновить unit-тесты в internal/kb под новый формат

## 3. internal/api

- [x] 3.1 Проверить handlers: если читают annotation/content/metadata напрямую — обновить под GetNode (возвращает те же поля)
- [x] 3.2 Обновить API-тесты handlers_test.go под новый формат узлов

## 4. cmd/kb-cli

- [x] 4.1 Обновить `kb-cli init`: при создании примера узла создавать `{dirname}.md` с frontmatter вместо annotation.md, content.md, metadata.json

## 5. Документация и skills

- [x] 5.1 Обновить `.cursor/skills/knowledge-db/SKILL.md`: структура узла (node-name.md с frontmatter)
- [x] 5.2 Обновить AGENTS.md и README при необходимости (структура data/)
