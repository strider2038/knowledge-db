## Why

Текущее ранжирование placement candidates поощряет крупные темы и использует формат источника как предметный сигнал. Из-за этого широкие ветки вроде `ai/agentic-coding` сами усиливают свой приоритет и поглощают узлы про RAG, память агентов, модели и онлайн-сервисы.

## What Changes

- Убрать положительный бонус за размер темы из предметного ranking score.
- Передавать в compact theme map сбалансированную выборку тем, а не только самые крупные каталоги.
- Ограничить placement query компактным набором значимых терминов вместо первых 80 слов сырого материала.
- Сделать `source_kind` и `content_profile` диагностическими, а не предметными сигналами темы.
- Уточнить prompt: `theme_path` описывает предмет, candidate themes являются подсказками, а широкая родительская тема не должна выбираться при наличии точной дочерней ветки.
- Добавить regression-тесты на типичные ошибки размещения.

## Capabilities

### New Capabilities

Нет.

### Modified Capabilities

- `ingestion-pipeline`: изменяются правила построения placement context и выбора `theme_path`.

## Impact

- `internal/ingestion/placement_builder.go` и тесты placement builder.
- `internal/ingestion/llm/prompt.go` и тесты prompt.
- OpenSpec capability `ingestion-pipeline`.
- API и frontmatter contract не меняются.
