## Purpose

Pipeline добавления записей в базу: текст или URL → создание узла с файлами. В scaffold — интерфейс и заглушка.

## Requirements

### Requirement: Интерфейс Ingester

Система ДОЛЖНА (SHALL) предоставлять интерфейс Ingester с методами IngestText(text string) и IngestURL(url string).

#### Сценарий: Вызов IngestText

- **WHEN** вызывается IngestText с текстом
- **THEN** система возвращает результат (в scaffold — ошибку "not implemented" или создаёт минимальный узел)

#### Сценарий: Вызов IngestURL

- **WHEN** вызывается IngestURL с URL
- **THEN** система возвращает результат (в scaffold — ошибку "not implemented" или создаёт минимальный узел)

### Requirement: Заглушка в scaffold

В scaffold реализация MUST возвращать ошибку "not implemented" или создавать минимальный узел без вызова LLM.

#### Сценарий: Заглушка при IngestText

- **WHEN** вызывается IngestText в scaffold
- **THEN** либо возвращается ошибка, либо создаётся узел без LLM-обработки
