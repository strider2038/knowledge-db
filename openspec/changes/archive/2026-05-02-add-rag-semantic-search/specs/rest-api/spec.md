## ADDED Requirements

### Requirement: Endpoint чатбота POST /api/chat

API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/chat` для RAG-чатбота. Запрос MUST содержать `message` (string). Ответ MUST быть streaming (SSE). При `KB_EMBEDDING_ENABLED=false` MUST возвращать 503. При пустом `message` MUST возвращать 400.

#### Scenario: Успешный запрос

- **WHEN** `POST /api/chat` с `{ "message": "..." }`
- **THEN** возвращается SSE stream с источниками и токенами ответа

#### Scenario: Сервис недоступен

- **WHEN** `POST /api/chat` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503

### Requirement: Управление индексом

API ДОЛЖЕН (SHALL) предоставлять endpoints для управления индексом: `POST /api/index/rebuild` — полная перестройка индекса (запускает SyncWorker ManualRebuild); `GET /api/index/status` — состояние индекса (total_nodes, total_chunks, embedding_model, last_indexed_at, status). Оба endpoint MUST возвращать 503 при `KB_EMBEDDING_ENABLED=false`.

#### Scenario: Запуск перестройки индекса

- **WHEN** `POST /api/index/rebuild`
- **THEN** запускается полная переиндексация, возвращается 202 Accepted

#### Scenario: Проверка статуса индекса

- **WHEN** `GET /api/index/status`
- **THEN** возвращается JSON с метриками индекса
