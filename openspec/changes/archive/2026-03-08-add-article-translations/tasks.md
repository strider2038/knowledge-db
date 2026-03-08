## 1. Конфигурация

- [x] 1.1 Добавить KB_AUTO_TRANSLATE в config (bool, default true)
- [x] 1.2 Передавать autoTranslate в PipelineIngester при создании

## 2. internal/kb: Store и утилиты

- [x] 2.1 Реализовать Store.CreateTranslationFile(ctx, basePath, themePath, slug, lang, frontmatter, content) — запись файла {themePath}/{slug}.{lang}.md
- [x] 2.2 Реализовать Store.AppendTranslationsToOriginal(ctx, basePath, themePath, slug, translationSlug string) — чтение оригинала, добавление translations в frontmatter и wikilink в тело, перезапись
- [x] 2.3 Реализовать функцию NeedsTranslation(content string) bool — эвристика: удаление code blocks, подсчёт кириллицы, порог 0.25, минимум 200 символов
- [x] 2.4 Тесты на NeedsTranslation: английский текст, русский текст, короткий текст, смешанный с code blocks

## 3. internal/ingestion/llm: перевод

- [x] 3.1 Реализовать TranslateToRussian(ctx, content string) (string, error) — Responses API без tools, Input + Instructions, извлечение текста из output items
- [x] 3.2 Тесты на TranslateToRussian с мок-клиентом

## 4. internal/ingestion/translation: чанкинг и оркестрация

- [x] 4.1 Реализовать разбиение на чанки: удаление code blocks (сохранение в буфер), разбиение по абзацам ~4000 символов, порог 6000
- [x] 4.2 Реализовать склейку: конкатенация переводов чанков, проверка дубликатов на стыках, вставка code blocks обратно
- [x] 4.3 Реализовать Translator.Translate(ctx, content) — вызов TranslateToRussian; при длине >6000 — чанкинг, перевод по частям, склейка
- [x] 4.4 Тесты на чанкинг и склейку

## 5. internal/ingestion: интеграция в pipeline

- [x] 5.1 Добавить в PipelineIngester поле autoTranslate и зависимость Translator
- [x] 5.2 Реализовать maybeTranslateAndSave: после saveNode при result.Type=="article", NeedsTranslation и autoTranslate — перевод, CreateTranslationFile, AppendTranslationsToOriginal, git commit
- [x] 5.3 Вызывать maybeTranslateAndSave из IngestText и IngestURL после успешного saveNode
- [x] 5.4 Тесты на pipeline с переводом (мок Translator)

## 6. internal/kb: валидация переводов

- [x] 6.1 Реализовать парсер wikilinks — извлечение [[target]] и [[target|label]] из markdown
- [x] 6.2 Расширить Store.Validate: проход по файлам *.ru.md, проверка frontmatter (translation_of, lang), проверка существования оригинала, проверка wikilinks на существующие узлы, проверка translations в оригинале
- [x] 6.3 Тесты на валидацию переводов

## 7. cmd/kb-cli: validate

- [x] 7.1 Убедиться, что validate вызывает Store.Validate (уже использует kb.Validate) — расширенная валидация должна применяться автоматически
- [x] 7.2 Тесты kb-cli validate с базой, содержащей переводы (валидные и невалидные)

## 8. Bootstrap и wiring

- [x] 8.1 Добавить KB_AUTO_TRANSLATE в config.Load
- [x] 8.2 Создавать Translator (LLM-based) в buildIngester при наличии LLM-конфигурации
- [x] 8.3 Передавать autoTranslate и Translator в NewPipelineIngester
