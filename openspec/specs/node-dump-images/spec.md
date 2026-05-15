# node-dump-images Specification

## Purpose
TBD - created by archiving change add-node-dump-images-action. Update Purpose after archive.
## Requirements
### Requirement: Операция dump images для одного узла

Система SHALL предоставлять серверный сценарий `dump images` для конкретного узла по path. Операция MUST запускаться асинхронно и возвращать диагностируемый статус выполнения.

#### Scenario: Успешный старт операции
- **WHEN** клиент запрашивает запуск `dump images` для существующего article-узла
- **THEN** сервер создаёт операцию со статусом `running` и возвращает её идентификатор

#### Scenario: Узел не найден
- **WHEN** клиент запрашивает запуск для несуществующего path
- **THEN** сервер возвращает `not found` и MUST NOT запускать операцию

### Requirement: Логирование операции dump images

Система SHALL собирать логи операции `dump images` в формате, эквивалентном логам нормализации: записи `stdout`, `stderr` и `system` с монотонным `offset` и временной меткой. API чтения логов MUST поддерживать инкрементальный режим по `after`.

#### Scenario: Инкрементальное чтение логов
- **WHEN** клиент запрашивает логи операции с `after=<offset>`
- **THEN** сервер возвращает только записи с offset больше указанного и `next_offset`

#### Scenario: Ошибка операции
- **WHEN** операция завершается ошибкой
- **THEN** финальные лог-записи и итоговый статус ошибки доступны через API логов/статуса

### Requirement: Автоматический sync после успешного dump images

После успешного завершения `dump images` система SHALL автоматически запускать `sync` и MUST возвращать итог операции с учётом результата sync.

#### Scenario: Dump и sync успешны
- **WHEN** `dump images` завершился success и `sync` прошёл успешно
- **THEN** итоговый статус операции `success`

#### Scenario: Dump успешен, sync завершился ошибкой
- **WHEN** основной шаг `dump images` завершился success, но `sync` вернул ошибку
- **THEN** операция возвращает ошибку шага sync с диагностикой

