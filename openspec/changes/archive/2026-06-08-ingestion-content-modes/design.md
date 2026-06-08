## Context

После change `add-link-profile-digests` (ADR 0010) ingestion научился генерировать digest для link/note и обновлять profile-link через refresh. Однако ось «как обрабатывать тело узла» осталась неявной: решение принимает LLM по противоречивому промпту, а код усиливает только часть сценариев (`ensureArticleContent` для article, `ensureDigestContent` только на refresh link).

Пользовательские каналы (Telegram, Add UI, import session) передают смешанный ввод: готовый текст, URL, инструкции, type hint. Без явного content mode система ошибочно интерпретирует paste как «скачай статью с URL» или «сделай digest».

Концептуальная документация: `docs/concepts/ingestion-workflows.md`.

## Goals / Non-Goals

**Goals:**

- Развести четыре workflow: verbatim capture, full fetch article, digest, link bookmark/profile.
- Сделать выбор режима предсказуемым и тестируемым без полной зависимости от LLM.
- Сохранить совместимость с `type`, `source_kind`, `content_profile` из ADR 0010.
- Закрыть класс регрессий из debug issues (paste+URL, Telegram verbatim, пустой digest, title noise).
- Симметричные guardrails для ingest и refresh.

**Non-Goals:**

- Массовый backfill существующих узлов.
- Новый обязательный `type` вместо `article|link|note`.
- Полный отказ от LLM для placement/metadata (keywords, theme_path, annotation).
- Автоматическая очистка YouTube-scrape в уже сохранённых узлах.

## Decisions

### Decision 1: Две оси — storage form и content mode

**Storage form** (существующая ось, ADR 0010):

- `type`: `article` | `link` | `note`
- `source_kind`, `content_profile`: природа источника и форма digest

**Content mode** (новая ось, вычисляется до LLM):

| Mode | Смысл | Типичный `type` | Тело |
|------|-------|-----------------|------|
| `verbatim` | Сохранить предоставленный пользователем текст | `note` (реже `article`) | Исходный markdown без переписывания |
| `full_fetch` | Полная копия с URL | `article` | Fetch по `source_url` |
| `digest` | Концептуальная/профильная выжимка | `note` или `link` | Структурированный digest по шаблону `content_profile` |
| `link_bookmark` | Минимальная закладка | `link` | Короткое semantic body из URL/meta/source text |

`content_mode` не дублирует `content_profile`: profile описывает **шаблон digest**, mode описывает **откуда брать тело и можно ли его переписывать**.

Инвариант сохранения: persisted `content` узла **всегда непустой**. Это нужно для семантического поиска и RAG. `link_bookmark` не означает пустой файл: он означает компактное тело без full-fetch и без развёрнутого профильного digest.

`content_mode` является операционным параметром ingest. Он передаётся в request, возвращается в response и пишется в logs/job logs, но **не сохраняется во frontmatter** в рамках этого change.

### Decision 2: Детерминированный resolver до LLM

Новая функция `ResolveContentMode(input, classification) ContentMode` в `internal/ingestion`:

Приоритет (сверху вниз):

1. Явный `content_mode` из API (`verbatim|full_fetch|digest|link_bookmark`) — если не `auto`.
2. Текстовые маркеры намерения: «сохрани полную статью» → `full_fetch`; «выжимка/концептуально/digest» → `digest`; «как есть/без изменений» → `verbatim`.
3. `TypeHint=article` **и** вход содержит только URL/короткий префикс → `full_fetch`.
4. `TypeHint=article` **и** вход содержит существенное тело (порог, напр. ≥500 символов или ≥80 слов вне URL) → `verbatim` (или `article` с телом из ввода, без fetch).
5. Telegram/delivery URL (`t.me`) + длинный текст → `verbatim` по умолчанию.
6. Только URL без тела → `full_fetch` при `TypeHint=article`; `digest` для источников с профильным digest (`repository_profile`, `documentation_profile`, `conceptual_digest` и т.п.); `link_bookmark` для неизвестных или минимальных закладок.
7. Fallback: `digest` для классифицированных внешних источников с digest/profile, `link_bookmark` для URL без профиля, `verbatim` для чистого текста.

Resolver записывает resolved mode в `ProcessInput`, response и logs. `ProcessInput` должен хранить два разных поля: исходное пользовательское тело (`RawContent` / `OriginalText`) и текст для LLM с системными префиксами. Verbatim guardrail использует только исходное пользовательское тело, а не префиксированный prompt text.

`TypeHint` управляет storage form (`article|link|note`), но не разрешает перезаписывать тело. При конфликте `content_mode` и `TypeHint` mode управляет обработкой тела, а `TypeHint` только фиксирует итоговый `type`, если это возможно без нарушения body-guardrail. Например, `content_mode=verbatim` + `type_hint=article` создаёт `type=article` с телом из paste и без fetch-замены.

### Decision 3: Единый post-LLM entrypoint, guardrails зависят от mode

После LLM pipeline вызывает общий `applyContentModeGuardrails(ctx, mode, rawContent, result)`. Этот entrypoint делегирует mode-specific helpers (`ensureArticleContent`, `ensureModeContent`, restore verbatim body), но место вызова одно и то же для ingest и refresh.

| Mode | Post-LLM правило |
|------|------------------|
| `verbatim` | `result.Content` = извлечённое тело из входа (не из fetch, не digest LLM). LLM генерирует только metadata/placement. |
| `full_fetch` | `ensureArticleContent` заполняет body из fetch/cache; исходный `RawContent` может быть инструкцией/контекстом, но не защищает body от замены |
| `digest` | `ensureModeContent` обязателен на ingest **и** refresh для `link` profile и `note` digest profiles |
| `link_bookmark` | `ensureModeContent` создаёт короткое semantic body; пустой content недопустим |

`ensureModeContent` — helper внутри общего entrypoint для modes, где LLM должен создать body. Он заменяет узкий смысл `ensureDigestContent`: проверяет непустое body и делает retry с mode-specific инструкцией. Для `digest` retry требует структурированный digest по `content_profile`; для `link_bookmark` retry требует компактное описание ресурса из доступных фактов (title, URL host/path, metadata, README/content preview, source text). Если фактов мало, body всё равно должно быть честным и минимальным, без домыслов.

### Decision 4: Упростить роль LLM по mode

- `verbatim`: LLM **не** вызывает `fetch_url_content` для тела; может вызывать `fetch_url_meta` только для annotation/keywords если есть `source_url`.
- `full_fetch`: LLM должен установить `source_url`; тело подставляет код из fetch/cache. Если fetch недоступен и нет уже полученного полного тела, ingest должен завершиться ошибкой, а не сохранять пустой body.
- `digest`: текущий tool flow с meta/content preview.
- `link_bookmark`: LLM использует `fetch_url_meta`, если доступен URL, и создаёт короткое body для поиска; `fetch_url_content` не вызывается ради full-copy.
- Промпт получает секцию «Content mode: …» с однозначными инструкциями.

Убрать из промпта конфликт: для `verbatim` note — «сохрани content из входа»; digest-правила применяются только при `content_mode=digest`.

### Decision 5: Нормализация title в коде

После `create_node`, в `saveNode` / `applyResultToExistingFrontmatter`:

- `stripMarkdownFromTitle` (уже есть)
- `normalizeTitleDecorators`: перенос leading emoji/символов в конец title (правило из markdown-normalization)
- Применять к `title` и единственному `aliases[0]`

Не полагаться на LLM для очистки заголовков каналов.

### Decision 6: API и UI

`POST /api/ingest` сохраняет существующие поля `text`, `source_url`, `source_author`, `type_hint` и принимает опциональное поле `content_mode`:

- `auto` (default), `verbatim`, `full_fetch`, `digest`, `link_bookmark`

Omitted `content_mode` трактуется как `auto`. Неизвестное непустое значение `content_mode` возвращает HTTP 400 с короткой ошибкой `invalid content_mode`; в отличие от legacy `type_hint`, оно не приводится молча к `auto`, потому что явный mode меняет обработку тела.

Успешный ответ `POST /api/ingest` — envelope:

```json
{
  "node": { "...": "kb.Node JSON" },
  "content_mode": "verbatim"
}
```

`content_mode` в ответе — resolved value после `auto`, не persisted frontmatter.

Web Add page: selector «Режим сохранения»:

- Авто
- Как есть (verbatim)
- Полная статья с URL (full_fetch)
- Выжимка / профиль (digest)
- Закладка (link_bookmark)

Существующий `type_hint` сохраняется для совместимости, но UI не должен предлагать `article` как единственный способ «вставить готовый текст».

В UI control «Режим сохранения» является primary для обработки тела. Control «Тип контента» остаётся secondary/advanced storage hint. UI должен явно показывать, что «Как есть + Статья» означает сохранить вставленный текст как `type=article`, а не скачать URL.

Import session использует тот же selector режима, что и ручное добавление. `POST /api/import/telegram/session/{id}/accept` принимает `content_mode` с теми же значениями и возвращает envelope `{ "node": ..., "next_item": ..., "content_mode": "..." }`.

### Decision 7: fetch_url_meta и GitHub README

При `digest` + repository URL: если `GitHubMetaFetcher` вернул README preview, **не** перезаписывать его более коротким Jina fallback (сейчас Jina всегда побеждает). Выбирать более информативный preview по длине/источнику.

### Decision 8: Refresh mode выводится из сохранённого узла

`content_mode` не persisted, поэтому `refresh-description` выводит mode из сохранённых полей. Body emptiness — repair trigger после выбора mode, а не первичный selector:

| Stored node | Refresh mode | Body rule |
|-------------|--------------|-----------|
| `type=article` + `source_url` | `full_fetch` | обновить/восстановить полную статью |
| `type=link` + `content_profile` кроме пустого/`link_bookmark` | `digest` | обновить структурированный profile digest |
| `type=link` + пустой/`link_bookmark` profile | `link_bookmark` | обновить компактное semantic body |
| `type=note` + `content_profile=conceptual_digest|brief_digest` + `source_url` | `digest` | обновить note digest |
| `type=note` без digest profile | `verbatim` | не переписывать body; обновлять только metadata, если это безопасно |

Если существующий узел имеет пустой body, это repair case. При наличии `source_url` refresh создаёт body по inferred mode. Без `source_url` refresh завершается понятной ошибкой и не должен генерировать содержание из воздуха. `type=article` без `source_url` сохраняет существующее source-url-required поведение refresh и не мутирует узел.

Telegram live bot и import session используют resolver auto-режима. Inline-кнопки Telegram для смены режима не входят в обязательный scope этого change. MCP ingestion tool сейчас отсутствует и остаётся out of scope; если такой tool появится позже, он должен использовать тот же enum и response contract.

## Risks / Trade-offs

- [Риск] Эвристики resolver ошибутся на пограничных кейсах. → [Mitigation] явный `content_mode` в API/UI/import; логирование выбранного mode; тесты на матрицу сценариев.
- [Риск] `verbatim` ухудшит RAG для длинных статей. → [Mitigation] пользователь явно выбирает режим; digest остаётся default для «только URL».
- [Риск] Дублирование логики с `TypeHint`. → [Mitigation] документировать приоритеты; постепенно сдвинуть UI на `content_mode`.
- [Риск] Увеличение сложности pipeline. → [Mitigation] mode как отдельный тип + таблица переходов; не плодить ветвления в промпте.

## Migration Plan

- Новые ingest используют resolver с `auto`; URL-only сценарии получают `full_fetch`, `digest` или `link_bookmark` по таблице resolver.
- Существующие узлы не мигрируются; refresh-description выводит mode из stored `type`/`content_profile`/`source_url`/body и применяет `ensureModeContent`.
- `content_mode` не добавляется во frontmatter в этом change.
- Rollback: откат кода; markdown в KB остаётся в git.

## References

- `docs/concepts/ingestion-workflows.md`
- ADR 0004, ADR 0010
- Debug issues: hermes-desktop-doklad, gemma-4, httptrace, plagin-bezopasnosti
