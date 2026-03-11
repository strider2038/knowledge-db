## 1. Синхронизация git-операций

- [ ] 1.1 Реализовать обёртку над GitCommitter с сериализацией вызовов CommitNode (канал или mutex + очередь)
- [ ] 1.2 Подключить обёртку в bootstrap: оборачивать ExecGitCommitter и передавать в pipeline и воркер

## 2. Атомарная запись файлов

- [ ] 2.1 Добавить атомарную запись в kb.CreateTranslationFile (запись во временный файл + os.Rename)
- [ ] 2.2 Добавить атомарную запись в kb.AppendTranslationsToOriginal (аналогично)

## 3. In-memory очередь переводов

- [ ] 3.1 Реализовать TranslationQueue: map по ключу themePath/slug, статусы pending/in_progress/done/failed, mutex
- [ ] 3.2 Реализовать Enqueue с дедупликацией: не создавать новую задачу при pending/in_progress
- [ ] 3.3 Реализовать GetStatus для получения статуса по ключу статьи

## 4. Интеграция с ingestion-пайплайном

- [ ] 4.1 Добавить TranslationQueue в параметры PipelineIngester
- [ ] 4.2 Обновить maybeTranslateAndSave: вместо вызова Translator.Translate вызывать queue.Enqueue(themePath, slug)

## 5. Фоновый воркер перевода

- [ ] 5.1 Реализовать TranslationWorker (runnable): чтение из очереди, смена статусов
- [ ] 5.2 Подключить Translator, Store, GitCommitter в воркер; вызывать Translate, CreateTranslationFile, AppendTranslationsToOriginal, CommitNode
- [ ] 5.3 Добавить context.WithTimeout для вызова LLM, обработать ошибки (статус failed + сообщение)
- [ ] 5.4 Зарегистрировать TranslationWorker через pior/runnable в bootstrap (при включённом LLM и не отключённом git)

## 6. REST API для перевода статьи

- [ ] 6.1 Добавить endpoint POST /api/articles/{id}/translate: поиск статьи по id, проверка наличия *.ru.md и очереди, Enqueue при необходимости
- [ ] 6.2 Добавить endpoint GET /api/articles/{id}/translate: возврат статуса (none/pending/in_progress/done/failed)
- [ ] 6.3 Добавить API-тесты для POST и GET /api/articles/{id}/translate

## 7. Web UI: пользовательский флоу перевода

- [ ] 7.1 Добавить кнопку «Перевести» на экране статьи (когда перевода нет)
- [ ] 7.2 Реализовать вызов POST при клике и polling GET до получения done/failed
- [ ] 7.3 Отображать индикацию «перевод в процессе» и ошибку при статусе failed

## 8. Наблюдаемость

- [ ] 8.1 Добавить логирование ключевых событий: enqueue (pending), start (in_progress), success (done), failure (failed)
