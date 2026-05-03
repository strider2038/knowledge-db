## ADDED Requirements

### Requirement: Endpoint гибридного поиска POST /api/search

API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/search` для гибридного поиска по базе знаний. Запрос MUST содержать `query` (string). Запрос MAY содержать `type`, `path`, `recursive`, `manual_processed`, `limit`, `offset` и `mode`. Ответ MUST содержать `results`, `total`, `query`, `mode` и метаданные retrieval. Endpoint MUST возвращать 503, если индекс недоступен для гибридного поиска.

#### Scenario: Успешный гибридный поиск

- **WHEN** клиент отправляет `POST /api/search` с `{ "query": "sqlite vector search" }`
- **THEN** API возвращает JSON со списком ранжированных карточек нод и релевантных фрагментов

#### Scenario: Пустой запрос

- **WHEN** клиент отправляет `POST /api/search` с пустым `query`
- **THEN** API возвращает 400 с ошибкой валидации

#### Scenario: Фильтр по типу

- **WHEN** клиент отправляет `POST /api/search` с `type=["article"]`
- **THEN** API возвращает только article-ноды

#### Scenario: Индекс недоступен

- **WHEN** `KB_EMBEDDING_ENABLED=false` или индекс не инициализирован
- **THEN** `POST /api/search` возвращает 503

## MODIFIED Requirements

### Requirement: Endpoint чатбота POST /api/chat

API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/chat` для RAG-чатбота. Запрос MUST содержать `message` (string). Запрос MAY содержать `source_paths` для ограничения ответа выбранными источниками. Ответ MUST быть streaming (SSE) и MUST использовать гибридный retrieval pipeline для поиска контекста. При `KB_EMBEDDING_ENABLED=false` MUST возвращать 503. При пустом `message` MUST возвращать 400.

#### Scenario: Успешный запрос

- **WHEN** `POST /api/chat` с `{ "message": "..." }`
- **THEN** выполняется гибридный retrieval, возвращается SSE stream с источниками и токенами ответа

#### Scenario: Запрос по выбранным источникам

- **WHEN** `POST /api/chat` содержит `source_paths`
- **THEN** контекст ответа ограничивается указанными источниками

#### Scenario: Сервис недоступен

- **WHEN** `POST /api/chat` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503

### Requirement: Управление индексом

API ДОЛЖЕН (SHALL) предоставлять endpoints для управления индексом: `POST /api/index/rebuild` — полная перестройка индекса (запускает SyncWorker ManualRebuild); `GET /api/index/status` — состояние индекса (total_nodes, total_chunks, embedding_model, keyword_index, last_indexed_at, status). Оба endpoint MUST возвращать 503 при `KB_EMBEDDING_ENABLED=false`.

#### Scenario: Запуск перестройки индекса

- **WHEN** `POST /api/index/rebuild`
- **THEN** запускается полная переиндексация, возвращается 202 Accepted

#### Scenario: Проверка статуса индекса

- **WHEN** `GET /api/index/status`
- **THEN** возвращается JSON с метриками индекса, включая режим keyword_index
