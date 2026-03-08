# Design: UI страницы узла — технические решения

## Context

Текущая NodePage показывает: кнопку «Назад» (Link to="/"), переключатель языков (если есть translations), заголовок = path, карточки «Аннотация» и «Содержание» с `<pre>`, карточку «Метаданные» с `JSON.stringify`. API возвращает Node: path, annotation, content, metadata (Record<string, unknown>).

Proposal требует: навигацию «Назад» с сохранением контекста, breadcrumbs по path, header с метаданными (без отдельного блока), markdown-рендеринг с таблицами и подсветкой кода.

## Goals / Non-Goals

**Goals:**
- Кнопка «Назад» возвращает на обзор с теми же query-параметрами
- Breadcrumbs: path → сегменты со ссылками на обзор с path-фильтром
- Header: title, type badge, created, updated, source_url (иконка + tooltip), source_author, source_date, keywords (чипсы)
- Аннотация и content — markdown с таблицами и подсветкой кода
- Метаданные не показывать отдельным блоком

**Non-Goals:**
- Редактирование узла
- Collapsible «Подробнее» для метаданных
- Подсветка кода для всех языков (базовый набор)

## Decisions

### 1. Навигация «Назад»

**Решение:** При переходе из OverviewPage передавать `state: { returnTo: location.pathname + location.search }` в Link. NodePage при клике «Назад»:
- если `location.state?.returnTo` — `navigate(location.state.returnTo)`
- иначе — `navigate('/')`

**Реализация:** OverviewPage: `<Link to={...} state={{ returnTo: location.pathname + location.search }}>`. NodePage: `<Button onClick={() => navigate(location.state?.returnTo ?? '/')}>` — при прямом заходе state отсутствует, fallback на `/`.

**Альтернатива:** `navigate(-1)` — при прямом заходе может вернуть на внешний сайт (если открыли по ссылке).

### 2. Breadcrumbs

**Решение:** Path разбить по `/`. Каждый сегмент — ссылка на Обзор с `?path=<сегмент>` (накопленный путь от корня). Например, path `programming/scaling/load-balancing`:
- `programming` → `/?path=programming`
- `scaling` → `/?path=programming/scaling`
- `load-balancing` → текущая страница (не ссылка, или ссылка на обзор с `?path=programming/scaling` для фильтра)

**Уточнение:** Последний сегмент — текущий узел. Варианты:
- A) Все сегменты — ссылки на обзор: `programming` → `/?path=programming`, `scaling` → `/?path=programming/scaling`, `load-balancing` → `/?path=programming/scaling/load-balancing` (последний откроет обзор, отфильтрованный по поддереву этого узла)
- B) Последний — текст без ссылки (текущая страница)

**Выбор:** A — все сегменты ссылки. При клике на последний — переход на обзор с path=полный путь, что покажет узлы в этом поддереве. Пользователь может «вернуться» на уровень выше.

**Формат:** `Обзор > programming > scaling > load-balancing` или `programming / scaling / load-balancing` с «Обзор» и сегментами как ссылками.

### 3. Header с метаданными

**Решение:** Одна компактная область под заголовком:

| Элемент | Источник | Отображение |
|---------|----------|-------------|
| Title | metadata.title \|\| slug(path) | h1 |
| Type | metadata.type | Badge (article=синий, link=зелёный, note=серый — как в Overview) |
| Created | metadata.created | Форматированная дата (toLocaleDateString) |
| Updated | metadata.updated | Форматированная дата |
| Source URL | metadata.source_url | Иконка ExternalLink, href=url, target="_blank"; Tooltip при hover с полным URL |
| Source author | metadata.source_author | Текст «Автор: X» |
| Source date | metadata.source_date | Форматированная дата «Дата источника: X» |
| Keywords | metadata.keywords | Массив чипсов (rounded, pill-style) |

**Порядок:** Type badge, created, updated, source_url (иконка), source_author, source_date, keywords. Разделитель — «·» или пробел.

**Условные:** source_url, source_author, source_date — только для article/link; keywords — показывать всегда (если есть).

### 4. Markdown-рендеринг

**Решение:** `react-markdown` + `remark-gfm` (таблицы, strikethrough, autolinks) + `rehype-highlight` (подсветка кода) или `prism-react-renderer` через rehype.

**Библиотеки:**
- `react-markdown` — основной рендерер
- `remark-gfm` — GFM (GitHub Flavored Markdown): таблицы, strikethrough, autolinks, task lists
- `rehype-highlight` — подсветка через highlight.js (легковесный, без дополнительных языков по умолчанию) или `rehype-prism` — если нужен Prism

**Рекомендация:** `rehype-highlight` + `highlight.js` — стандартный набор, поддерживает основные языки. Стили — тема из highlight.js (например, github-dark для dark mode).

**Компонент:** `MarkdownContent({ content }: { content: string })` — обёртка над react-markdown с кастомными компонентами для ссылок (target="_blank", rel="noopener noreferrer"), заголовков (классы для стилей).

### 5. Структура страницы

```
┌─────────────────────────────────────────────────────────────┐
│ [← Назад]                                                    │
┌─────────────────────────────────────────────────────────────┤
│ Обзор > programming > scaling > load-balancing               │
├─────────────────────────────────────────────────────────────┤
│ [Оригинал] [ru] [en]  (если есть translations)               │
├─────────────────────────────────────────────────────────────┤
│ Title (h1)                                                   │
│ [article] · 8 мар 2026 · 8 мар 2026 · [🔗] · Автор: X ·      │
│ Дата источника: 8 мар 2026 · [keyword1] [keyword2] [keyword3]│
├─────────────────────────────────────────────────────────────┤
│ Аннотация                                                    │
│ (markdown)                                                   │
├─────────────────────────────────────────────────────────────┤
│ Содержание                                                   │
│ (markdown)                                                   │
└─────────────────────────────────────────────────────────────┘
```

**Карточки:** Аннотация и Содержание — Card с CardHeader/CardContent. Содержание может быть пустым (для link) — показывать «(нет)» или не показывать блок.

### 6. Зависимости

**Добавить в web/package.json:**
- `react-markdown` — рендеринг markdown
- `remark-gfm` — GFM расширения
- `rehype-highlight` — подсветка кода
- `highlight.js` — языки для подсветки (peer или bundled)

**Альтернатива:** `rehype-prism` + `prism-react-renderer` — более гибкий, но тяжелее. Начать с rehype-highlight.

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| XSS в markdown (если пользовательский контент) | react-markdown по умолчанию санитизирует; не использовать dangerouslySetInnerHTML |
| highlight.js увеличивает bundle | Подключить только нужные языки (javascript, typescript, bash, json, yaml, python, go) |
| Ссылки в markdown открываются в той же вкладке | Кастомный компонент `a` с target="_blank" rel="noopener noreferrer" |

## Migration Plan

1. Добавить зависимости: react-markdown, remark-gfm, rehype-highlight, highlight.js
2. Создать компонент MarkdownContent
3. OverviewPage: добавить state в Link
4. NodePage: переработать — breadcrumbs, header, кнопка «Назад», MarkdownContent для annotation и content
5. Удалить блок «Метаданные»
6. Убедиться, что Node из API содержит metadata.title, metadata.source_url и т.д. (уже есть)

Роллбек: revert коммитов.

## Open Questions

(Нет открытых вопросов.)
