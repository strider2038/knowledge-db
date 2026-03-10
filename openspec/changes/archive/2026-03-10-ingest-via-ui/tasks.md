## 1. Backend: TypeHint в ingestion

- [x] 1.1 Добавить поле TypeHint (string) в IngestRequest (internal/ingestion/ingester.go)
- [x] 1.2 Добавить поле TypeHint в ProcessInput (internal/ingestion/llm/orchestrator.go)
- [x] 1.3 Обновить buildProcessInput в pipeline: передавать req.TypeHint в ProcessInput
- [x] 1.4 Обновить buildSystemPrompt: при TypeHint = article/link/note добавлять инструкцию «Пользователь указал тип: <type>. Используй именно этот тип при вызове create_node»
- [x] 1.5 Обновить StubIngester: принимать IngestRequest с TypeHint (обратная совместимость)

## 2. API: приём type_hint

- [x] 2.1 Добавить парсинг type_hint в handler Ingest (internal/api/handlers.go): опциональное поле, допустимые значения auto/article/link/note
- [x] 2.2 Передавать TypeHint в IngestRequest при вызове IngestText
- [x] 2.3 Добавить API-тест: POST /api/ingest с type_hint передаёт значение в Ingester

## 3. Web: переключатель типа и UX

- [x] 3.1 Добавить переключатель типа (авто, статья, ссылка, заметка) в AddPage
- [x] 3.2 При выборе «статья» или «ссылка» показывать подсказку «Вставьте URL в текст»
- [x] 3.3 Обновить ingestText в api.ts: принимать type_hint, возвращать Node с path
- [x] 3.4 Добавить/использовать спиннер при loading: анимация загрузки
- [x] 3.5 Блокировать textarea и кнопку при loading, текст кнопки «Обработка...»
- [x] 3.6 Успех: Alert/Banner с «Добавлено» и ссылкой «Перейти к узлу» (использовать path из ответа)
- [x] 3.7 Ошибка: Alert/Banner с текстом ошибки

## 4. Тесты и проверки

- [x] 4.1 Обновить internal/ingestion/pipeline_test.go: вызовы IngestText с IngestRequest (TypeHint)
- [x] 4.2 Добавить тест оркестратора с TypeHint (internal/ingestion/llm/orchestrator_test.go при необходимости)
