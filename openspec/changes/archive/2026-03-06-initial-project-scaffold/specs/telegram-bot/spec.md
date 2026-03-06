## Purpose

Telegram-бот для приёма сообщений и URL, вызова ingestion pipeline. Работает в том же процессе, что и kb-server.

## Requirements

### Requirement: Конфигурация через env

Бот ДОЛЖЕН (SHALL) читать TELEGRAM_TOKEN и KB_DATA_PATH из переменных окружения.

#### Сценарий: Запуск без токена

- **WHEN** kb-server запущен без TELEGRAM_TOKEN
- **THEN** бот не стартует или логирует предупреждение

### Requirement: Приём сообщений

Бот MUST принимать текстовые сообщения и URL от пользователей.

#### Сценарий: Получение текста

- **WHEN** пользователь отправляет боту текст
- **THEN** бот передаёт текст в ingestion pipeline (в scaffold — заглушка)

#### Сценарий: Получение URL

- **WHEN** пользователь отправляет боту ссылку
- **THEN** бот передаёт URL в ingestion pipeline (в scaffold — заглушка)

### Requirement: Long polling

Бот ДОЛЖЕН (SHALL) использовать long polling для получения обновлений (без публичного URL).

#### Сценарий: Работа без webhook

- **WHEN** бот запущен
- **THEN** он получает обновления через long polling, без необходимости настраивать webhook
