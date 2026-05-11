## MODIFIED Requirements

### Requirement: Endpoint чатбота
API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/chat` для RAG-чатбота с поддержкой чат-сессий. Запрос MUST содержать `message` и идентификатор активной сессии (или признак создания новой сессии). Запрос MAY содержать список `source_paths` для ограничения ответа выбранными источниками из поиска. Ответ MUST быть streaming (Server-Sent Events) с источниками, найденным контекстом (при наличии) и токенами ответа LLM. Генерация ответа SHOULD использовать OpenAI-compatible Chat Completions streaming, чтобы работать с локальными провайдерами вроде LM Studio. Endpoint MUST возвращать 503 при `KB_EMBEDDING_ENABLED=false`.

#### Scenario: Успешный запрос к чатботу в существующей сессии
- **WHEN** `POST /api/chat` отправлен с `session_id` и `{ "message": "Какие паттерны DI в Go?" }`
- **THEN** выполняется retrieval по базе, учитывается контекст сессии, и LLM генерирует ответ как SSE stream

#### Scenario: Запрос с выбранными источниками
- **WHEN** `POST /api/chat` содержит `source_paths`
- **THEN** retrieval и контекст ответа ограничиваются указанными нодами и их фрагментами

#### Scenario: Пустое сообщение
- **WHEN** `POST /api/chat` с пустым или отсутствующим `message`
- **THEN** возвращается 400 с описанием ошибки валидации

#### Scenario: Embeddings отключены
- **WHEN** `POST /api/chat` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503

#### Scenario: Streaming совместим с LM Studio
- **WHEN** chat provider настроен на OpenAI-compatible `/v1` endpoint локальной модели
- **THEN** backend использует chat completions streaming и stream-ит delta content как SSE token events

#### Scenario: SSE не буферизуется gzip middleware
- **WHEN** браузер или клиент отправляет `Accept-Encoding: gzip`
- **THEN** `/api/chat` response не сжимается gzip и содержит headers, разрешающие немедленную доставку SSE chunks

#### Scenario: Превышение лимита истории сессии
- **WHEN** контекст активной сессии превышает лимит prompt budget
- **THEN** система MUST использовать summary-сегменты и последние сообщения, сохраняя непрерывность диалога без превышения лимита

### Requirement: Серверная память диалога
Чат ДОЛЖЕН (SHALL) обрабатывать `POST /api/chat` с серверной памятью активной сессии. Backend MUST строить prompt из текущего `message`, выбранных `source_paths`, найденного RAG-контекста и истории активной сессии (включая summary-сегменты при необходимости).

#### Scenario: Следующий вопрос получает историю активной сессии
- **WHEN** пользователь отправляет несколько сообщений подряд в одном чате
- **THEN** backend использует предыдущие user/assistant сообщения активной сессии как conversation history

## RENAMED Requirements

### Requirement: Отсутствие серверной памяти диалога
- FROM: `Отсутствие серверной памяти диалога`
- TO: `Серверная память диалога`
