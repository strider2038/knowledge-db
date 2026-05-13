## 1. Placement Context Model

- [ ] 1.1 Добавить внутренние структуры `PlacementContext`, `ThemeCandidate`, `KeywordCandidate`, `SimilarNode` в ingestion/llm или отдельный пакет ingestion placement.
- [ ] 1.2 Расширить `llm.ProcessInput` полем placement context, сохранив совместимость существующих тестов на время миграции.
- [ ] 1.3 Добавить лимиты candidate themes, candidate keywords и similar nodes константами.

## 2. Candidate Builder

- [ ] 2.1 Реализовать builder, который извлекает поисковые сигналы из входного текста, source metadata, `source_kind`, `content_profile` и fetched preview.
- [ ] 2.2 Реализовать fallback-обход файлов базы для сбора theme profiles, similar nodes и keyword statistics без локального индекса.
- [ ] 2.3 Подключить локальный index store при доступности для поиска похожих узлов и curated vocabulary.
- [ ] 2.4 Реализовать scoring candidate themes с учётом похожих узлов, совпадений title/annotation/keywords/path, source profile и плотности темы.
- [ ] 2.5 Реализовать scoring candidate keywords с учётом входных терминов, keywords похожих узлов, top keywords candidate themes и частотности, без ручного словаря синонимов.

## 3. Prompt And Tools

- [ ] 3.1 Обновить системный prompt: заменить секции полного списка themes/keywords на краткую карту базы, candidate themes, candidate keywords и similar nodes.
- [ ] 3.2 Добавить tool `search_placement_candidates(query, source_kind, content_profile, type)` в schema tools.
- [ ] 3.3 Реализовать обработку tool call `search_placement_candidates` в function calling loop без изменения финального `create_node`.
- [ ] 3.4 Обновить инструкции prompt: LLM должна предпочитать candidate themes/keywords, вызывать уточняющий tool только при сомнении и не создавать синонимы без необходимости.

## 4. Pipeline Integration And Diagnostics

- [ ] 4.1 Подключить placement builder в `PipelineIngester.buildProcessInput` перед первым вызовом LLM.
- [ ] 4.2 Добавить fallback-поведение при ошибках индекса: продолжать ingestion через файловый builder и логировать причину.
- [ ] 4.3 Добавить диагностические логи: количество candidate themes, candidate keywords, similar nodes, источник candidates (`index` или `fallback`), размер старого и нового prompt context.
- [ ] 4.4 Убедиться, что явные пользовательские инструкции вроде `сохрани в go/concurrency` имеют приоритет над автоматическим shortlist.

## 5. Tests

- [ ] 5.1 Добавить unit-тесты builder для ранжирования тем на пересекающихся ветках `ai/agentic-coding`, `ai/agentic-coding/skills`, `programming/ai`.
- [ ] 5.2 Добавить unit-тесты keyword candidates для ранжирования частотных и тематически близких keywords без ручной дедупликации синонимов.
- [ ] 5.3 Обновить prompt tests: prompt содержит placement context и не содержит полный список всех keywords как основной механизм.
- [ ] 5.4 Добавить orchestrator tests для `search_placement_candidates` tool call и продолжения loop до `create_node`.
- [ ] 5.5 Добавить pipeline tests для fallback без индекса и логически достаточного placement context.

## 6. Verification

- [ ] 6.1 Запустить Go-тесты для `internal/ingestion/...`, `internal/ingestion/llm/...` и затронутых index/helper пакетов.
- [ ] 6.2 Проверить OpenSpec status для change и убедиться, что apply-ready.
- [ ] 6.3 Провести ручную пробу на локальной базе с материалом про agent skills/Claude Code и убедиться, что candidates включают релевантные ветки.
