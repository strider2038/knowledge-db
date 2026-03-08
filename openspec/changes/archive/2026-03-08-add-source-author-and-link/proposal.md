# Proposal: Сохранение автора и ссылки на источник

## Why

При сохранении контента в базу знаний важно сохранять атрибуцию: кто автор и откуда взят материал. Сейчас при пересылке поста из Telegram теряются ссылка на оригинал и автор; при сохранении статей (Habr и др.) автор извлекается ContentFetcher, но не попадает в frontmatter узла. Без этого невозможно корректно цитировать источники и восстанавливать контекст.

## What Changes

- **Пересланные сообщения Telegram**: при пересылке поста бот MUST извлекать из `forward_origin` автора (имя канала, username пользователя или `author_signature`) и ссылку на оригинал (https://t.me/...), передавать их в ingestion и сохранять в узле.
- **Статьи из интернета (Habr и др.)**: ContentFetcher уже возвращает `Author` в FetchResult; pipeline MUST передавать автора в create_node и сохранять в frontmatter узла как `source_author`.
- **Хранение**: добавить опциональное поле `source_author` в frontmatter узла (knowledge-storage). Поле `source_url` уже есть.

## Capabilities

### New Capabilities

(нет — меняем существующие)

### Modified Capabilities

- `knowledge-storage`: добавить опциональное поле `source_author` в frontmatter (автор источника: имя, username, название канала и т.п.)
- `telegram-bot`: при пересланном сообщении извлекать из `forward_origin` автора и ссылку, передавать их в ingestion (расширить контекст, передаваемый в IngestText, или добавить метаданные в текст)
- `ingestion-pipeline`: добавить `source_author` в tool create_node; при вызове fetch_url_content использовать Author из FetchResult; сохранять source_author в frontmatter при create_node

## Impact

- **internal/kb**: CreateNodeParams — добавить SourceAuthor в Frontmatter при наличии
- **internal/telegram**: парсинг ForwardOrigin (JSON), извлечение author/link, передача в ingestion
- **internal/ingestion**: ProcessInput — опциональные поля SourceURL, SourceAuthor; pipeline — передача в ProcessInput; orchestrator — source_author в create_node, подстановка Author из FetchResult; saveNode — source_author в frontmatter
- **internal/ingestion/llm**: prompt — описание source_author в create_node; buildTools — добавить source_author в schema
