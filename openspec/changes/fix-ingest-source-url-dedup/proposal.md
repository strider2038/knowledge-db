## Why

При ingestion заметки с `source_url`, совпадающим с уже существующим узлом, пайплайн безусловно обновляет найденный узел вместо создания нового. Это привело к потере данных: аудит двух agent skills перезаписал старый узел `ai/agentic-coding/taste-skill`, хотя пользователь ожидал новую заметку. Дедупликация по `source_url` задумывалась для повторного импорта той же статьи/ссылки, но ошибочно применяется к `type: note` и свободному тексту, где `source_url` — лишь метаданные об объекте, а не идентификатор «тот же ресурс».

## What Changes

- Ограничить автодедупликацию по нормализованному `source_url` типами `article` и `link`; для `type: note` всегда создавать новый узел, если явно не передан `node_id`.
- Сохранить update по явному `node_id` для всех типов контента.
- Сохранить дедуп по `source_url` для повторного `IngestURL` / re-fetch статьи и закладки-ссылки.
- Добавить регрессионный тест на сценарий «заметка с `source_url`, совпадающим с существующим узлом».
- Уточнить спеки `node-identity` и `ingestion-pipeline` под новое правило.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `node-identity`: уточнить правило дедупликации по `source_url` — только для `article` и `link`.
- `ingestion-pipeline`: уточнить create/update при сохранении узла; добавить сценарии для заметок с совпадающим `source_url`.

## Impact

- `internal/ingestion/save_node_dedup.go` — условие lookup по `source_url`.
- `internal/ingestion/pipeline_test.go` — новый тест и корректировка существующих при необходимости.
- `openspec/specs/node-identity/spec.md`, `openspec/specs/ingestion-pipeline/spec.md` — после архивации change.
- REST API, web UI, индекс SQLite — без изменения контрактов; поведение ingestion меняется только для заметок с `source_url`.
