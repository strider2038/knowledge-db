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
| `digest` | Концептуальная/профильная выжимка | `note` или `link` | LLM digest по шаблону `content_profile` |
| `link_bookmark` | Минимальная закладка | `link` | Пустое или краткое |

`content_mode` не дублирует `content_profile`: profile описывает **шаблон digest**, mode описывает **откуда брать тело и можно ли его переписывать**.

### Decision 2: Детерминированный resolver до LLM

Новая функция `ResolveContentMode(input) ContentMode` в `internal/ingestion`:

Приоритет (сверху вниз):

1. Явный `content_mode` из API (`verbatim|full_fetch|digest|link_bookmark`) — если не `auto`.
2. Текстовые маркеры намерения: «сохрани полную статью» → `full_fetch`; «выжимка/концептуально/digest» → `digest`; «как есть/без изменений» → `verbatim`.
3. `TypeHint=article` **и** вход содержит только URL/короткий префикс → `full_fetch`.
4. `TypeHint=article` **и** вход содержит существенное тело (порог, напр. ≥500 символов или ≥80 слов вне URL) → `verbatim` (или `article` с телом из ввода, без fetch).
5. Telegram/delivery URL (`t.me`) + длинный текст → `verbatim` по умолчанию.
6. Только URL без тела → `digest` или `full_fetch` по `TypeHint` и классификации источника (как сейчас).
7. Fallback: `digest` для классифицированных внешних источников, `verbatim` для чистого текста.

Resolver записывает mode в `ProcessInput` и в текстовый префикс для LLM.

### Decision 3: Guardrails после LLM зависят от mode

| Mode | Post-LLM правило |
|------|------------------|
| `verbatim` | `result.Content` = извлечённое тело из входа (не из fetch, не digest LLM). LLM генерирует только metadata/placement. |
| `full_fetch` | `ensureArticleContent` только если content пустой/усечён; **не** перезаписывать непустое тело из ввода |
| `digest` | `ensureDigestContent` обязателен на ingest **и** refresh для `link` profile и `note` digest profiles |
| `link_bookmark` | Пустое тело допустимо |

### Decision 4: Упростить роль LLM по mode

- `verbatim`: LLM **не** вызывает `fetch_url_content` для тела; может вызывать `fetch_url_meta` только для annotation/keywords если есть `source_url`.
- `full_fetch`: LLM должен установить `source_url` и пустой content; тело подставляет код.
- `digest`: текущий tool flow с meta/content preview.
- Промпт получает секцию «Content mode: …» с однозначными инструкциями.

Убрать из промпта конфликт: для `verbatim` note — «сохрани content из входа»; digest-правила применяются только при `content_mode=digest`.

### Decision 5: Нормализация title в коде

После `create_node`, в `saveNode` / `applyResultToExistingFrontmatter`:

- `stripMarkdownFromTitle` (уже есть)
- `normalizeTitleDecorators`: перенос leading emoji/символов в конец title (правило из markdown-normalization)
- Применять к `title` и единственному `aliases[0]`

Не полагаться на LLM для очистки заголовков каналов.

### Decision 6: API и UI

`POST /api/ingest` принимает опциональное поле `content_mode`:

- `auto` (default), `verbatim`, `full_fetch`, `digest`, `link_bookmark`

Web Add page: selector «Режим сохранения»:

- Авто
- Как есть (verbatim)
- Полная статья с URL (full_fetch)
- Выжимка / профиль (digest)
- Закладка (link_bookmark)

Существующий `type_hint` сохраняется для совместимости, но UI не должен предлагать `article` как единственный способ «вставить готовый текст».

### Decision 7: fetch_url_meta и GitHub README

При `digest` + repository URL: если `GitHubMetaFetcher` вернул README preview, **не** перезаписывать его более коротким Jina fallback (сейчас Jina всегда побеждает). Выбирать более информативный preview по длине/источнику.

## Risks / Trade-offs

- [Риск] Эвристики resolver ошибутся на пограничных кейсах. → [Mitigation] явный `content_mode` в API/UI; логирование выбранного mode; тесты на матрицу сценариев.
- [Риск] `verbatim` ухудшит RAG для длинных статей. → [Mitigation] пользователь явно выбирает режим; digest остаётся default для «только URL».
- [Риск] Дублирование логики с `TypeHint`. → [Mitigation] документировать приоритеты; постепенно сдвинуть UI на `content_mode`.
- [Риск] Увеличение сложности pipeline. → [Mitigation] mode как отдельный тип + таблица переходов; не плодить ветвления в промпте.

## Migration Plan

- Новые ingest используют resolver с `auto`; поведение для «только URL» остаётся близким к текущему digest/full_fetch.
- Существующие узлы не меняются; refresh-description работает по новым guardrails после реализации.
- Rollback: откат кода; markdown в KB остаётся в git.

## References

- `docs/concepts/ingestion-workflows.md`
- ADR 0004, ADR 0010
- Debug issues: hermes-desktop-doklad, gemma-4, httptrace, plagin-bezopasnosti
