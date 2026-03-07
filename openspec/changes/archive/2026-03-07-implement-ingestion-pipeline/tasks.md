## 1. Конфигурация

- [x] 1.1 Добавить новые поля в config: LLM_API_URL, LLM_API_KEY, LLM_MODEL, JINA_API_KEY (опционально), GIT_SYNC_INTERVAL (default "5m"), TELEGRAM_OWNER_ID
- [x] 1.2 Валидация: TELEGRAM_OWNER_ID обязателен при наличии TELEGRAM_TOKEN; если нет LLM-конфигурации — warning в лог

## 2. Расширение kb: frontmatter и Store.CreateNode

- [x] 2.1 Расширить frontmatter: добавить опциональные поля type, source_url, source_date. Валидация не ломается — старые узлы без этих полей остаются валидными
- [x] 2.2 Реализовать Store.CreateNode(ctx, basePath, CreateNodeParams) — создание директории узла, файла {slug}.md с frontmatter + content, автосоздание промежуточных директорий
- [x] 2.3 Обработка slug-коллизий: суффикс -2, -3 при совпадении имени
- [x] 2.4 Тесты на CreateNode: создание в существующей теме, в новой теме, коллизия slug

## 3. ContentFetcher

- [x] 3.1 Определить интерфейс ContentFetcher и структуру FetchResult (Title, Content, SourceDate, Author)
- [x] 3.2 Реализовать JinaFetcher — GET https://r.jina.ai/{url}, парсинг ответа в FetchResult
- [x] 3.3 Реализовать ReadabilityFetcher — HTTP GET + go-readability + html-to-markdown, добавить зависимости go-shiori/go-readability и JohannesKaufmann/html-to-markdown
- [x] 3.4 Реализовать ChainFetcher — Jina primary, ReadabilityFetcher fallback при ошибке
- [x] 3.5 Реализовать функцию fetch_url_meta — извлечение title + description из HTML <meta> тегов (легковесный HTTP GET + парсинг <head>)
- [x] 3.6 Тесты на ChainFetcher: успешный Jina, fallback при ошибке Jina, оба недоступны

## 4. LLM-оркестратор

- [x] 4.1 Добавить зависимость github.com/openai/openai-go/v3
- [x] 4.2 Определить интерфейс LLMOrchestrator и структуры ProcessInput / ProcessResult
- [x] 4.3 Реализовать LLM-клиент: Responses API (client.Responses.New), определение tools (fetch_url_content, fetch_url_meta, create_node), обработка function calling loop
- [x] 4.4 Составить системный промпт: роль, правила выбора type, инструкции по использованию существующих тем и keywords
- [x] 4.5 Реализовать сбор контекста базы: список существующих тем (Store.ReadTree) и keywords (обход узлов)
- [x] 4.6 Тесты на LLM-оркестратор с мок-клиентом: сценарии note, article, link

## 5. GitCommitter

- [x] 5.1 Определить интерфейс GitCommitter (CommitNode, Sync)
- [x] 5.2 Реализовать ExecGitCommitter — git add + commit через exec.Command
- [x] 5.3 Реализовать Sync — git fetch origin + git merge origin/main, логирование конфликтов
- [x] 5.4 Реализовать GitSyncRunner (runnable) — периодический вызов Sync с конфигурируемым интервалом
- [x] 5.5 Тесты на CommitNode и Sync (с git-репозиторием в tmp)

## 6. PipelineIngester

- [x] 6.1 Реализовать PipelineIngester: IngestText (вызов LLMOrchestrator.Process → Store.CreateNode → GitCommitter.CommitNode)
- [x] 6.2 Реализовать IngestURL (явный вызов ContentFetcher.Fetch + LLM для метаданных + CreateNode + Commit)
- [x] 6.3 Тесты на PipelineIngester с мок-зависимостями

## 7. Telegram-бот

- [x] 7.1 Добавить авторизацию: проверка from.id == TELEGRAM_OWNER_ID в handleUpdate, игнорирование с warning для посторонних
- [x] 7.2 Добавить sendMessage — POST /sendMessage для ответа пользователю
- [x] 7.3 Расширить handleUpdate: вызов IngestText, отправка подтверждения (путь, type, keywords) или ошибки
- [x] 7.4 Тесты на авторизацию и обработку сообщений

## 8. Bootstrap и wiring

- [x] 8.1 Создание PipelineIngester с реальными зависимостями в bootstrap.Run() вместо StubIngester (при наличии LLM-конфигурации)
- [x] 8.2 Fallback на StubIngester при отсутствии LLM-конфигурации с warning
- [x] 8.3 Регистрация GitSyncRunner как runnable
- [x] 8.4 Передача TELEGRAM_OWNER_ID в telegram.Bot

## 9. API тесты

- [x] 9.1 API тест POST /api/ingest — успешный ingestion (с мок-ingester)
- [x] 9.2 API тест POST /api/ingest — ошибка ingestion
