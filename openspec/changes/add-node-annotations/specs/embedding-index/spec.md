## Purpose

Дельта: явное исключение sidecar аннотаций из embedding-индекса.

## ADDED Requirements

### Requirement: annotations.yaml не участвует в индексе

Содержимое файла `{slug}/annotations.yaml` MUST NOT читаться SyncWorker для построения embedding text, searchable text, chunks или content_hash. Изменение только аннотаций MUST NOT запускать переиндексацию узла.

#### Сценарий: Добавление личной заметки

- **WHEN** пользователь добавляет или изменяет записи только в `annotations.yaml`
- **THEN** `content_hash` и `body_hash` узла в indexed_nodes не меняются

#### Сценарий: Индексация узла с sidecar

- **WHEN** SyncWorker индексирует узел, у которого есть `annotations.yaml`
- **THEN** текст аннотаций не попадает в embedding и searchable text
