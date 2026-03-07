## Why

Ingestion pipeline существует только как интерфейс и заглушка (stub 501). Нет способа добавить контент в базу знаний — ни через Telegram, ни через API, ни программно. Это блокирует использование системы по назначению.

## What Changes

- Реализация `Ingester` — полноценная обработка текста и URL вместо заглушки
- Парсинг URL: извлечение контента статей для оффлайн-хранения (Jina Reader API + Go-native readability fallback)
- LLM-генерация метаданных: keywords, annotation, тема/подтема, тип контента — автоматически при добавлении
- Расширение frontmatter: новые поля `type` (article/link/note), `source_url`, `source_date`
- Запись нод: создание директории узла и файлов в `data/` + git commit
- Telegram-бот: определение типа входящего сообщения (URL, текст, пересылка), вызов соответствующего метода ingester, подтверждение пользователю

## Capabilities

### New Capabilities

_(нет новых — все покрываются расширением существующих)_

### Modified Capabilities

- `ingestion-pipeline`: из заглушки в полноценный pipeline — парсинг URL (Jina + fallback), LLM-генерация метаданных, создание узлов в файловой системе
- `knowledge-storage`: расширение frontmatter — добавление полей `type`, `source_url`, `source_date`; обновление валидации
- `telegram-bot`: определение типа сообщения (URL/текст/пересылка), передача в ingester с контекстом, ответ пользователю с подтверждением

## Impact

- **internal/ingestion/**: замена `StubIngester` на реальную реализацию с зависимостями (HTTP-клиент, LLM-клиент, kb.Store с записью)
- **internal/kb/**: расширение `Store` методами записи (CreateNode), расширение frontmatter и валидации
- **internal/telegram/**: логика определения типа сообщения, форматирование ответа
- **Новые зависимости**: HTTP-клиент для Jina Reader API, Go-библиотеки go-readability + html-to-markdown (fallback), LLM API клиент (OpenAI-совместимый)
- **API**: `POST /api/ingest` начнёт работать (перестанет возвращать 501)
- **Git**: ingester будет коммитить в data-репозиторий при добавлении записей
