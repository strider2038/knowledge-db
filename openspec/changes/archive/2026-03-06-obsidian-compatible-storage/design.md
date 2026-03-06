# Design: obsidian-compatible-storage

## Context

Текущий формат узла knowledge-db: папка с тремя файлами — `annotation.md`, `content.md`, `metadata.json`. Obsidian ожидает один .md файл на заметку с YAML frontmatter. Папка с несколькими .md даёт несколько отдельных заметок в Obsidian, а не одну. Нужна прямая совместимость без конвертера: vault = база knowledge-db, открывается в Obsidian как есть.

## Goals / Non-Goals

**Goals:**

- Один главный .md файл на узел — Obsidian видит одну заметку
- Метаданные в YAML frontmatter (совместимо с Obsidian)
- Сохранение иерархии тем (topic/subtopic)
- Сохранение notes/, images/, .local/

**Non-Goals:**

- Обратная совместимость со старым форматом (annotation.md + content.md + metadata.json) — в scope отдельного решения
- Конвертер между форматами — не нужен при прямом совпадении

## Decisions

### 1. Варианты организации узла (документация обоих)

Рассматривались два подхода к совместимости с Obsidian.

#### Вариант A: Папка на узел, один главный .md (выбран)

```
topic/
├── subtopic/
│   └── node-name/
│       ├── node-name.md     ← главная заметка (Obsidian + kb)
│       ├── notes/
│       ├── images/
│       └── .local/
```

- Узел = папка, главный файл = `{имя-папки}.md`
- Obsidian: одна заметка `node-name` в папке `topic/subtopic/node-name/`
- kb: читает node-name.md, парсит frontmatter и тело
- notes/, images/ — без изменений

**Плюсы:** привычная структура «узел = папка», notes и images рядом с главным файлом.

#### Вариант B: Один .md на узел, без папки узла

```
topic/
├── subtopic/
│   ├── node-name-1.md
│   ├── node-name-2.md
│   └── node-name-3/         ← папка только при наличии подзаметок
│       ├── node-name-3.md
│       └── notes/
```

- Простые узлы — один .md в папке темы
- Узлы с подзаметками — своя папка с главным .md и notes/

**Плюсы:** меньше вложенности для простых узлов. **Минусы:** смешанная модель (файл vs папка), сложнее логика IsNodeDir.

**Решение: Вариант A.** Сохраняем единообразие: узел всегда папка. Проще валидация и обход дерева.

### 2. Формат главного .md файла

**Структура:**

```yaml
---
source: https://...
sourceType: article
keywords: [llm, rag]
created: 2026-03-01T12:00:00Z
updated: 2026-03-01T12:00:00Z
annotation: "Краткое описание узла"
---

# Заголовок

Основной контент...
```

- **Frontmatter:** source (опционально), sourceType, keywords, created, updated, annotation
- **Тело:** markdown-контент (аналог content.md)
- Поле `annotation` в frontmatter заменяет annotation.md

### 3. Имя главного файла

**Правило:** главный файл = `{dirname}.md`, где dirname — имя папки узла.

Пример: папка `topic/subtopic/rag-overview/` → главный файл `rag-overview.md`.

Альтернатива `index.md` отклонена: в Obsidian заметка будет называться «index», что менее наглядно.

### 4. Парсинг frontmatter в Go

**Решение:** использовать библиотеку для YAML frontmatter, например `github.com/adrg/frontmatter` или аналог. Проверить наличие в go.mod; при отсутствии — добавить.

### 5. IsNodeDir: новая логика

**Было:** наличие annotation.md, content.md, metadata.json.

**Стало:** папка считается узлом, если содержит файл `{dirname}.md` с валидным frontmatter (keywords, created, updated).

Проверка: `os.Stat(filepath.Join(path, dirname+".md")) == nil` и парсинг frontmatter для валидации полей.

### 6. GetNode: чтение из одного .md

**Было:** читать annotation.md, content.md, metadata.json отдельно.

**Стало:** читать `{dirname}.md`, парсить frontmatter → Metadata + Annotation, тело → Content.

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| Существующие базы в старом формате | Миграционный скрипт или отдельный change для обратной совместимости |
| Невалидный YAML во frontmatter | Валидация при GetNode, понятная ошибка пользователю |
| Поле annotation опционально | Спека: annotation рекомендуется, но не обязателен; content (тело) обязателен |

## Migration Plan

1. Реализовать новый формат в internal/kb
2. Обновить kb-cli init — создавать node-name.md вместо annotation.md, content.md, metadata.json
3. Миграция существующих баз: отдельная утилита или ручной скрипт (конвертация annotation+content+metadata → один .md с frontmatter)
4. Обновить agent skill и документацию
