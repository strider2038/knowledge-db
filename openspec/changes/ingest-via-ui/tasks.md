## 1. Backend: TypeHint в ingestion

- [ ] 1.1 Добавить поле TypeHint (string) в IngestRequest (internal/ingestion/ingester.go)
- [ ] 1.2 Добавить поле TypeHint в ProcessInput (internal/ingestion/llm/orchestrator.go)
- [ ] 1.3 Обновить buildProcessInput в pipeline: передавать req.TypeHint в ProcessInput
- [ ] 1.4 Обновить buildSystemPrompt: при TypeHint = article/link/note добавлять инструкцию «Пользователь указал тип: <type>. Используй именно этот тип при вызове create_node»
- [ ] 1.5 Обновить StubIngester: принимать IngestRequest с TypeHint (обратная совместимость)

## 2. API: приём type_hint

- [ ] 2.1 Добавить парсинг type_hint в handler Ingest (internal/api/handlers.go): опциональное поле, допустимые значения auto/article/link/note
- [ ] 2.2 Передавать TypeHint в IngestRequest при вызове IngestText
- [ ] 2.3 Добавить API-тест: POST /api/ingest с type_hint передаёт значение в Ingester

## 3. Web: переключатель типа и UX

- [ ] 3.1 Добавить переключатель типа (авто, статья, ссылка, заметка) в AddPage
- [ ] 3.2 При выборе «статья» или «ссылка» показывать подсказку «Вставьте URL в текст»
- [ ] 3.3 Обновить ingestText в api.ts: принимать type_hint, возвращать Node с path
- [ ] 3.4 Добавить/использовать спиннер при loading: анимация загрузки
- [ ] 3.5 Блокировать textarea и кнопку при loading, текст кнопки «Обработка...»
- [ ] 3.6 Успех: Alert/Banner с «Добавлено» и ссылкой «Перейти к узлу» (использовать path из ответа)
- [ ] 3.7 Ошибка: Alert/Banner с текстом ошибки

## 4. Тесты и проверки

- [ ] 4.1 Обновить internal/ingestion/pipeline_test.go: вызовы IngestText с IngestRequest (TypeHint)
- [ ] 4.2 Добавить тест оркестратора с TypeHint (internal/ingestion/llm/orchestrator_test.go при необходимости)
