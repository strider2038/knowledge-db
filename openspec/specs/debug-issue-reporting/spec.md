## Purpose

Локальное файловое хранение отладочных артефактов (issue-репорты из web UI, опциональные raw-логи Telegram) под `KB_DATA_PATH/.kb` и отдельный служебный MCP endpoint для чтения этих данных агентами, без смешивания с основным MCP базы знаний.

## Requirements

### Requirement: Файловое сохранение issue-репортов из web UI

Система ДОЛЖНА (SHALL) принимать bug report из web UI и сохранять его в `KB_DATA_PATH/.kb/issues/...` как отдельный Markdown-файл с frontmatter и текстовым описанием проблемы.

#### Scenario: Создание issue с NodePage

- **WHEN** пользователь отправляет багрепорт со страницы узла
- **THEN** система SHALL создать файл issue в `.kb/issues` с полями контекста страницы узла, включая путь узла в JSON контекста под ключом `nodePath` (camelCase в payload API, см. `web/src/pages/NodePage.tsx`)

#### Scenario: Создание issue с SearchPage

- **WHEN** пользователь отправляет багрепорт со страницы семантического поиска
- **THEN** система SHALL сохранить в issue параметры поиска, метаданные выдачи и список результатов, необходимых для воспроизведения

#### Scenario: Создание issue с ChatPage

- **WHEN** пользователь отправляет багрепорт со страницы чата
- **THEN** система SHALL сохранить полный контекст чата, включая полные сообщения сессии и связанные sources ответа

### Requirement: Статусный пайплайн issue и API обновления статуса

Система ДОЛЖНА (MUST) поддерживать статусы issue `new`, `investigating`, `fixed` и SHALL предоставлять API для обновления статуса существующего issue-файла.

#### Scenario: Статус по умолчанию при создании issue

- **WHEN** создается новый issue из web UI
- **THEN** система MUST записать в frontmatter статус `new`

#### Scenario: Обновление статуса issue

- **WHEN** клиент вызывает API обновления статуса с валидным идентификатором issue и статусом `investigating` или `fixed`
- **THEN** система SHALL обновить frontmatter issue-файла и вернуть обновленное состояние

#### Scenario: Невалидный статус

- **WHEN** клиент вызывает API обновления статуса со значением вне `new`, `investigating`, `fixed`
- **THEN** система MUST отклонить запрос с ошибкой валидации

### Requirement: Опциональный лог raw Telegram сообщений

Система ДОЛЖНА (MUST) поддерживать запись сырых Telegram update/message в `KB_DATA_PATH/.kb/telegram-raw/*.ndjson` при включенном флаге окружения и НЕ ДОЛЖНА писать такие логи при выключенном флаге.

#### Scenario: Логирование включено

- **WHEN** `KB_TELEGRAM_RAW_LOG_ENABLED=true` и бот получает сообщение
- **THEN** система SHALL добавить запись в дневной NDJSON-файл с исходным payload и техническими метаданными времени/идентификаторов

#### Scenario: Логирование выключено

- **WHEN** `KB_TELEGRAM_RAW_LOG_ENABLED=false` и бот получает сообщение
- **THEN** система MUST не создавать и не обновлять файлы в `.kb/telegram-raw`

### Requirement: TTL для raw Telegram логов 14 дней

Система ДОЛЖНА (SHALL) автоматически удалять raw Telegram лог-файлы старше 14 дней от текущего времени сервера.

#### Scenario: Удаление устаревших файлов

- **WHEN** в `.kb/telegram-raw` есть файлы старше 14 дней
- **THEN** система MUST удалить такие файлы в рамках регулярной очистки

#### Scenario: Сохранение свежих файлов

- **WHEN** файл raw Telegram лога не старше 14 дней
- **THEN** система SHALL оставить файл без изменений

### Requirement: Отдельный debug MCP endpoint для служебных артефактов

Система ДОЛЖНА (SHALL) предоставлять отдельный MCP endpoint для debug-артефактов (issues и telegram raw), отделенный от основного MCP endpoint базы знаний.

#### Scenario: Debug MCP выключен

- **WHEN** `KB_MCP_DEBUG_API_KEY` не задан
- **THEN** система MUST не обслуживать debug MCP endpoint

#### Scenario: Чтение issues через debug MCP

- **WHEN** клиент вызывает debug MCP tool для списка/чтения issues с корректным debug API ключом
- **THEN** система SHALL вернуть данные из файлов `.kb/issues` в структурированном виде

#### Scenario: Чтение raw Telegram через debug MCP

- **WHEN** клиент вызывает debug MCP tool для чтения последних raw Telegram записей с корректным debug API ключом
- **THEN** система SHALL вернуть записи из `.kb/telegram-raw` без изменения исходного payload
