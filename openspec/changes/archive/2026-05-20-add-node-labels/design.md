## Context

В базе знаний уже есть `keywords` (семантика, RAG, обзорный поиск `q`) и `manual_processed` (workflow-флаг с фильтром API/UI). Пользователю нужны произвольные личные метки на узлах, фильтрация дерева на «Обзоре» (режим AND) и цветные чипы без отдельного конфиг-файла в репозитории.

Текущие точки расширения: `internal/kb` (frontmatter, `ListNodesWithOptions`), `PATCH /api/nodes`, `OverviewPage` (`treeFilterPaths`), `buildNodeEmbeddingText` в `internal/index/sync.go`.

## Goals / Non-Goals

**Goals:**

- Поле `labels: []string` во frontmatter узла (git-sync, только узлы).
- Нормализация при записи: trim, удаление пустых, дедупликация без учёта регистра (сохранять первое написание).
- API: чтение, PATCH, фильтр `GET /api/nodes?labels=a,b` (AND).
- UI: редактирование на странице узла; фильтр и чипы на «Обзоре»; сужение дерева по меткам (паттерн как `type` / `manual_processed`).
- Цвет метки: стабильный индекс палитры по hash строки метки (общий модуль `label-styles`, аналог `type-styles`).
- Исключить `labels` из embedding text и content_hash (смена меток не пересчитывает эмбеддинги).

**Non-Goals:**

- Отдельный `.kb/labels.yaml` или реестр меток.
- Obsidian-совместимое поле `tags`.
- Telegram-бот, MCP, bulk-скрипты.
- Семантический / hybrid search по меткам.
- Метки на папках-темах.
- OR-режим фильтра (только AND в этой итерации).

## Decisions

### 1. Хранение: `labels` во frontmatter

**Выбор:** опциональный массив строк в YAML узла.

**Альтернативы:** `.local/` (не в git), центральный JSON (merge-конфликты, move path). Frontmatter согласуется с git-first и паттерном `manual_processed`.

### 2. Отделение от `keywords`

**Выбор:** отдельное поле `labels`; не попадает в `q`, embedding, `content_hash`.

**Альтернатива:** префикс в keywords (`!favorite`) — легко попасть в RAG, отклонено.

### 3. Фильтр API: AND, query `labels`

**Выбор:** `GET /api/nodes?labels=foo,bar` — узел возвращается только если содержит **все** перечисленные метки (после нормализации сравнение case-insensitive).

Пустой параметр не передаётся. Один label — частный случай AND.

### 4. PATCH

**Выбор:** расширить существующий `PATCH /api/nodes/{path}` полем `labels` (массив строк, полная замена списка). Пустой массив удаляет ключ из frontmatter.

Не делать отдельный `POST .../labels/add` в MVP.

### 5. Цвета в UI

**Выбор:** фиксированная палитра из ~8–12 пар (bg/text/border, light+dark), индекс `fnv32(label) % len(palette)`. Без персистентного конфига.

**Альтернатива:** реестр в файле — отклонено по запросу пользователя.

### 6. Индекс SQLite

**Выбор:** не включать `labels` в embedding/searchable text для RAG. Опционально: колонка `labels` в `node_search` (JSON) **только** если понадобится ускорить фильтр на больших базах; MVP — фильтрация в `kb.Store` при обходе (как сегодня для части фильтров).

`content_hash` без `labels` — изменение меток не триггерит переиндексацию embedding.

### 7. Подсказки в UI

**Выбор:** `GET /api/labels` или переиспользовать агрегацию из существующего endpoint keyword suggestions — отдельный лёгкий endpoint `GET /api/label-suggestions` (уникальные labels по базе, лимит N). Избегать смешения с keyword suggestions.

### 8. Лимиты

- Максимум **32** метки на узел (валидация PATCH / store).
- Длина одной метки **64** символа после trim.
- Символы: печатные Unicode без управляющих; запятая в значении запрещена (разделитель query).

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| Дубли `Favorite` / `favorite` | Нормализация case-insensitive при сохранении и фильтре |
| Путаница keywords vs labels в UI | Разные подписи («Ключевые слова» / «Метки»), разные цвета чипов |
| Медленный фильтр дерева (запрос 10k узлов) | Тот же паттерн, что для type; позже — кэш в index |
| Засорение frontmatter | Лимиты длины и количества |

## Migration Plan

Обратная совместимость: узлы без `labels` → `labels: []` в JSON. Существующие API-клиенты не ломаются.

Деплой: server + embedded web; переиндексация не требуется (labels вне content_hash).

Откат: удалить поддержку поля; frontmatter `labels` останется в файлах без вреда.

## Open Questions

- _(нет блокирующих; решения зафиксированы в proposal)_
