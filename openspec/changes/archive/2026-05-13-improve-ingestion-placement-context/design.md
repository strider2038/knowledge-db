## Context

Ingestion pipeline сейчас собирает контекст базы через `ReadTree` и обход всех узлов: в prompt попадают все существующие темы и все уникальные keywords. На текущей пользовательской базе это уже около 50 тем и более 400 уникальных keywords. Размер сам по себе ещё умеренный, но контекст не ранжирован и не объясняет LLM, какие темы живые, какие keywords каноничные и какие варианты являются синонимами.

Особенно заметны пересечения между ветками `ai/*` и `programming/*`, а также дублирующие keywords вроде `AI`, `ИИ`, `искусственный интеллект`, `AI-агенты`, `ИИ-агенты`, `Go`, `Golang`. Из-за этого LLM иногда выбирает менее подходящую ветку или создаёт новый keyword вместо повторного использования существующего.

## Goals / Non-Goals

**Goals:**

- Сформировать перед первым LLM-запросом компактный `placement context`, который помогает выбрать `theme_path` и keywords.
- Передавать LLM candidate themes, candidate keywords и similar nodes вместо полного неранжированного списка keywords.
- Сохранить offline-first поведение: подбор кандидатов должен работать по локальной базе и локальному индексу, без сетевых зависимостей.
- Дать LLM tool для дополнительного поиска placement candidates, если первичный shortlist не покрывает материал.
- Сделать подбор диагностируемым: логировать количество кандидатов, причины ранжирования и размер prompt context.

**Non-Goals:**

- Не менять формат markdown/frontmatter.
- Не добавлять пользовательский UI для ручной настройки keyword-синонимов.
- Не вводить ручной словарь синонимов keywords и не решать обслуживание уже накопленных похожих терминов; чистка похожих keywords относится к отдельному процессу обслуживания базы.
- Не требовать embeddings для работы ingestion: векторный поиск может быть бонусом, но lexical/index/fallback путь обязателен.
- Не подключать векторный поиск в первой реализации placement builder; это отдельная оптимизация после lexical версии.
- Не решать массовую рекаталогизацию уже созданных узлов.

## Decisions

### 1. Основной механизм — preprocessing до первого LLM-запроса

Перед вызовом `LLMOrchestrator.Process` pipeline должен построить `PlacementContext`:

```go
type PlacementContext struct {
    ThemeMap          []ThemeSummary
    CandidateThemes   []ThemeCandidate
    CandidateKeywords []KeywordCandidate
    SimilarNodes      []SimilarNode
}
```

Решение: preprocessing выполняется всегда, когда pipeline готовит `ProcessInput`. Это даёт LLM качественный shortlist уже в первом запросе и уменьшает зависимость от того, решит ли модель вызвать tool.

Альтернатива: дать только tool `search_placement_candidates`. Отклонено как основной путь, потому что модель может не вызвать tool, вызвать его поздно или сформулировать плохой query.

### 2. Candidate themes ранжируются по локальным сигналам

Подбор тем должен учитывать:

- похожие узлы из локального индекса или fallback-поиска;
- совпадение title/annotation/keywords/path с входным текстом и fetched preview;
- совпадение `source_kind` и `content_profile`;
- плотность темы (`NodeCount`) и близость leaf-темы;
- parent/sibling контекст, чтобы LLM могла выбрать более общий или более точный путь.

Пример результата:

```text
1. ai/agentic-coding/skills
   reason: skills + agent tools + repository_profile
   examples: awesome-openclaw-skills, visual-explainer

2. ai/agentic-coding/claude-code
   reason: Claude Code specific materials

3. ai/agentic-coding
   reason: parent topic with many agentic coding notes
```

### 3. Candidate keywords строятся из кандидатов, а не из полного словаря

Keywords должны приходить из четырёх источников:

- термины из входного текста, title, URL/domain и preview;
- keywords похожих узлов;
- top keywords candidate themes;
- небольшой curated global vocabulary как fallback.

Каждый keyword-кандидат должен иметь сведения о частоте и темах использования. На уровне этой задачи builder не ведёт ручной словарь синонимов и не пытается обслуживать накопленный словарь базы. Допустима только техническая нормализация для scoring (регистр, пробелы, точное совпадение написания); смысловые дубли вроде `AI`/`ИИ` или `Go`/`Golang` должны решаться отдельным обслуживанием базы.

Альтернатива: оставить полный список keywords, но отсортировать. Это уменьшит недетерминизм, однако не решит проблему смыслового шума.

### 4. Tool остаётся уточняющим каналом

В LLM tools добавляется `search_placement_candidates(query, source_kind, content_profile, type)`. Tool возвращает те же структуры: candidate themes, candidate keywords, similar nodes. Prompt должен объяснять, что tool полезен при сомнении между ветками или если primary candidates не подходят.

Tool не должен быть обязательным для happy path. Финальное действие всё равно `create_node`.

### 5. Fallback без индекса обязателен

Если `.kb/index.db` отсутствует, не синхронизирован или недоступен, preprocessing должен работать через существующий `Store`: обход узлов, чтение frontmatter, построение простого lexical score по title, annotation, keywords и path. Это медленнее, но сохраняет offline-first и предсказуемость.

### 6. Совместимость ProcessInput

Можно временно оставить `ExistingThemes`/`ExistingKeywords` для обратной совместимости тестов, но новый prompt должен предпочитать `PlacementContext`. После миграции старые поля можно удалить или оставить как deprecated internal fields.

## Risks / Trade-offs

- [Risk] Неверный shortlist может сузить поле зрения LLM. → Mitigation: включать parent/sibling темы, краткую top-level карту базы и tool уточнения.
- [Risk] Fallback-обход файлов замедлит ingestion на больших базах. → Mitigation: использовать индекс при доступности, кешировать theme profiles в рамках процесса, ограничивать количество похожих узлов.
- [Risk] Похожие keywords в базе продолжат существовать и попадать в candidates. → Mitigation: считать дедупликацию словаря отдельной задачей обслуживания базы; в этой задаче только ранжировать уже существующие keywords и не добавлять ручной словарь синонимов.
- [Risk] Prompt может стать сложнее. → Mitigation: заменить длинный плоский список keywords на компактные структурированные секции с лимитами.
- [Risk] Tool увеличит число LLM-итераций. → Mitigation: считать tool уточняющим; primary preprocessing должен покрывать типичный случай без дополнительного вызова.

## Migration Plan

1. Добавить структуры placement context и builder с fallback-обходом файлов.
2. Подключить локальный индекс как источник similar nodes и curated vocabulary при доступности.
3. Обновить `ProcessInput` и prompt builder, сохранив старые поля на время миграции.
4. Добавить tool `search_placement_candidates` в function calling loop.
5. Покрыть тестами подбор кандидатов, prompt rendering и tool loop.
6. После стабилизации убрать или пометить deprecated старую передачу полного списка keywords.

Откат: вернуть prompt к `ExistingThemes`/`ExistingKeywords` и не регистрировать новый tool; формат данных и API при этом не меняются.

## Resolved Decisions

- Лимиты candidate themes, candidate keywords и similar nodes на первом этапе задаются константами, без новых env-переменных.
- Ручной словарь синонимов keywords не входит в scope. Проблема похожих терминов должна решаться обслуживанием базы и отдельной чисткой словаря.
- Векторный поиск не входит в первую реализацию placement builder. Первая версия использует lexical/index/fallback сигналы; vector search можно добавить позже как optional signal поверх lexical builder.
