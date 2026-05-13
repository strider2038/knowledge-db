## Why

Текущий ingestion-оркестратор получает в prompt полный список тем и полный список keywords, из-за чего при росте базы ухудшается соотношение сигнал/шум. На реальной базе уже видны пересечения тем и синонимичные keywords (`AI`/`ИИ`/`искусственный интеллект`, `Go`/`Golang`, `AI-агенты`/`ИИ-агенты`), поэтому LLM не всегда стабильно выбирает место узла и каноничные ключевые слова.

## What Changes

- Заменить передачу полного списка keywords в LLM prompt на компактный `placement context`: кандидатные темы, кандидатные keywords и похожие узлы.
- Сохранять обзор дерева тем, но подавать его как краткую карту базы и ранжированный shortlist, а не как неструктурированный глобальный каталог.
- Добавить детерминированный preprocessing перед первым LLM-запросом: подбор candidate themes/keywords по входному тексту, метаданным источника, `source_kind`, `content_profile`, локальному индексу или fallback-обходу файлов.
- Добавить инструмент уточнения для LLM-оркестратора, который возвращает placement candidates по дополнительному query, если первичного shortlist недостаточно.
- Нормализовать и ранжировать candidate keywords: учитывать частоты, привязку к темам и похожие узлы без введения ручного словаря синонимов.
- Логировать размер и состав placement context для диагностики качества и стоимости prompt.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `ingestion-pipeline`: LLM-оркестратор должен получать компактный placement context вместо полного неранжированного списка тем и keywords, а также иметь возможность уточнить кандидатов через tool.

## Impact

- Backend ingestion: `internal/ingestion`, `internal/ingestion/llm`.
- Поисковый индекс: возможно переиспользование `internal/index` для похожих узлов и curated vocabulary; должен быть fallback без готового индекса.
- Prompt/tool schema для LLM orchestration.
- Тесты ingestion/llm prompt, preprocessing и function calling loop.
- Внешние REST/API контракты не меняются; формат markdown/frontmatter не меняется.
