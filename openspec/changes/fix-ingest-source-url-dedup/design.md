## Context

Дедупликация ingestion реализована в `internal/ingestion/save_node_dedup.go`. Функция `resolveExistingNode` ищет узел по `node_id`, затем по нормализованному `source_url` через SQLite-индекс (`FindBySourceURL`). `saveNode` при любом совпадении вызывает `updateExistingNode`, перезаписывая frontmatter и body.

Инцидент (issue-20260614-061737): пользователь добавил заметку-аудит двух GitHub-репозиториев. LLM вернул `type: note` и `source_url: https://github.com/Leonxlnx/taste-skill`. В индексе уже был узел `ai/agentic-coding/taste-skill` с тем же URL. Пайплайн обновил старый узел вместо создания нового.

Текущие спеки (`node-identity`, `ingestion-pipeline`) описывают безусловный lookup по `source_url` без учёта `type`.

## Goals / Non-Goals

**Goals:**

- Предотвратить перезапись существующего узла при ingestion заметки (`type: note`) только из-за совпадающего `source_url`.
- Сохранить дедупликацию для `article` и `link` — повторный импорт той же страницы/закладки обновляет существующий узел.
- Сохранить явный update по `node_id`.
- Покрыть регрессионным тестом сценарий из инцидента.

**Non-Goals:**

- Полное отключение дедупликации по `source_url`.
- Изменение схемы индекса или frontmatter.
- UI-переключатель «создать новый / обновить существующий» (можно отдельным change).
- Изменение prompt LLM для выбора `source_url` (полезно, но не решает корневую проблему).

## Decisions

### 1. Ограничить lookup по `source_url` типами `article` и `link`

**Решение:** в `resolveExistingNode` перед `FindBySourceURL` проверять `result.Type`. Если тип пустой или равен `note`, lookup по URL MUST NOT выполняться (кроме явного `node_id`).

**Почему:** для `article`/`link` `source_url` семантически означает «канонический URL ресурса»; для `note` — «объект, о котором пишем», и один URL может фигурировать в множестве независимых заметок (аудит, сравнение, обзор).

**Альтернативы:**

- *Отключить дедуп полностью* — ломает re-fetch статьи.
- *Сравнивать title/slug* — хрупко, зависит от LLM.
- *Дедуп только в `IngestURL`* — не покрывает Telegram/API с `type_hint=article`, зато слишком узко для link.

### 2. Пустой `type` трактовать как «без дедупа по URL»

**Решение:** если LLM не вернул `type`, не применять lookup по `source_url` (только `node_id`).

**Почему:** безопаснее создать дубликат, чем перезаписать данные. Пустой type на практике редок после оркестратора.

### 3. Логирование пропуска дедупа

**Решение:** при пропуске lookup по URL из-за `type: note` писать `clog.Info` с `source_url` и `type`.

**Почему:** упрощает диагностику в production без изменения API.

### 4. Тест на регрессию

**Решение:** тест `IngestText` — существующий узел `type: article|link` с URL X; новый ingest `type: note` с тем же URL → создаётся второй узел, первый не меняется.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Дубликаты `article`/`link` с одним URL при ошибочном `type: note` от LLM | Отдельный change: prompt/валидация type; пользователь может явно указать `type_hint` |
| Повторная заметка «обновить мысли о репозитории» создаст дубликат | Ожидаемо; для update — refresh/re-ingest с `node_id` или ручное редактирование |
| Существующий тест дедупа для article не затронут | Оставить `TestPipelineIngester_IngestURL_WhenDuplicateSourceURL_ExpectSameIDAndPath` без изменений |

## Migration Plan

1. Задеплоить исправление в `internal/ingestion`.
2. Повреждённые узлы восстанавливаются из git history (вне scope кода).
3. Reindex не требуется — схема индекса не меняется.

## Open Questions

- Нужен ли в будущем API-флаг `force_create` для power users? (не в этом change)
- Стоит ли в prompt явно предупреждать LLM не ставить `source_url` репозитория в multi-subject заметках? (отдельная задача)
