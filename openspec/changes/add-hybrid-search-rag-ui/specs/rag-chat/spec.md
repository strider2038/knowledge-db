## MODIFIED Requirements

### Requirement: Endpoint чатбота

API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/chat` для RAG-чатбота. Запрос MUST содержать поле `message` (string, обязательно). Запрос MAY содержать список `source_paths` для ограничения ответа выбранными источниками из поиска. Ответ MUST быть streaming (Server-Sent Events) с источниками, найденным контекстом (при наличии) и токенами ответа LLM. Endpoint MUST возвращать 503 при `KB_EMBEDDING_ENABLED=false`.

#### Scenario: Успешный запрос к чатботу

- **WHEN** `POST /api/chat` с `{ "message": "Какие паттерны DI в Go?" }`
- **THEN** выполняется гибридный retrieval по базе, собирается контекст, LLM генерирует ответ как SSE stream

#### Scenario: Запрос с выбранными источниками

- **WHEN** `POST /api/chat` содержит `source_paths`
- **THEN** retrieval и контекст ответа ограничиваются указанными нодами и их фрагментами

#### Scenario: Пустое сообщение

- **WHEN** `POST /api/chat` с пустым или отсутствующим `message`
- **THEN** возвращается 400 с описанием ошибки валидации

#### Scenario: Embeddings отключены

- **WHEN** `POST /api/chat` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503

### Requirement: Контекстная сборка для RAG

Система ДОЛЖНА (SHALL) собирать контекст для LLM из результатов гибридного retrieval pipeline. Контекст SHALL формироваться из ранжированных нод и фрагментов, найденных через exact/keyword/FTS/vector совпадения. Общий размер контекста MUST ограничиваться (не более ~4000 токенов). Если найдены и ноды, и чанки одной и той же статьи, статья MUST быть представлена наиболее релевантными чанками и краткими метаданными ноды.

#### Scenario: Найдены релевантные ссылки и статьи

- **WHEN** hybrid retrieval нашёл 3 link-ноды по keywords и 2 chunk'а из статьи по semantic близости
- **THEN** контекст содержит annotations ссылок и фрагменты статьи, общим размером не более 4000 токенов

#### Scenario: Ничего релевантного не найдено

- **WHEN** hybrid retrieval не нашёл результатов выше chat cutoff
- **THEN** LLM получает пустой контекст или сервер stream-ит ответ о том, что информация в базе не найдена

#### Scenario: Приоритет точного совпадения

- **WHEN** запрос содержит точный термин из keywords
- **THEN** соответствующая нода включается в контекст даже при отсутствии высокого vector score

### Requirement: Источники в ответе

Ответ чатбота ДОЛЖЕН (SHALL) включать список источников — нод базы знаний, выбранных retrieval pipeline для формирования ответа. Каждый источник MUST содержать: path, title, type и MAY содержать fragments с heading/snippet. Источники SHALL отправляться как SSE event `data: {"sources": [...]}\n\n` перед началом токенов ответа. UI MAY отображать их как “найденные источники”, если LLM не выдаёт inline citations.

#### Scenario: Ответ с источниками

- **WHEN** чатбот формирует ответ на основе 3 нод
- **THEN** перед токенами ответа отправляется SSE event с sources: [{path, title, type, fragments}, ...]

#### Scenario: Ответ без релевантных нод

- **WHEN** hybrid retrieval не нашёл релевантных результатов
- **THEN** sources отправляется как пустой массив

#### Scenario: Источник с фрагментом статьи

- **WHEN** источником ответа является chunk статьи
- **THEN** source содержит fragment с heading и snippet, чтобы пользователь мог понять основание ответа
