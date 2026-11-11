# Implementation Slices

Срезы — границы покрытия одного исхода: агент узла переключается конфигом, Cursor остаётся умолчанием. Один исполнитель может закрыть их вместе.

## Skipped slices

- `tests` отдельным срезом — тесты входят в Acceptance и чекбоксы срезов, которым принадлежат.

## Slice: Агент узла на сервере

> **Type**: domain-api
> **Outcome**: `KB_NODE_AGENT` выбирает Cursor или OpenCode для нормализации и agent-edit; валидация на старте; 503 без бинаря; логи OpenCode и запрет записи вне файла узла.
> **Acceptance**: `go test ./internal/api/ ./internal/bootstrap/config/ ./internal/bootstrap/ -count=1` и `golangci-lint run ./internal/api/... ./internal/bootstrap/...`
> **Skills**: `kb-backend-golang`, `golang-errors`, `golang-tests`, `api-conventions`
> **Scope**: `internal/bootstrap/config`, `internal/bootstrap/bootstrap.go`, `internal/api` (раннеры, джобы, тесты)
> **Allowed fallout**: моки тестов, сигнатуры `nodeNormalizer` и `NodeAgentEditor`, текст ошибки 503
> **Blocked**: смена маршрутов, формата узла, ingestion LLM (`LLM_*`), git sync после джобы

- [x] 1.1 `ValidateNodeAgent`: пусто/`cursor` успешно; `opencode` требует модель; `opencode-go/*` требует `OPENCODE_API_KEY` и запрещает `KB_NODE_AGENT_API_URL`; свой URL требует ключ; иное значение — ошибка. Вызов из `bootstrap.Run`.
- [x] 1.2 Общий раннер с `Available()`. Cursor сохраняет argv и текст `cursor-agent not found in PATH`. Type assert удалён.
- [x] 1.3 OpenCode: `run --standalone --format json --archived`, изолированные XDG, конфиг вне `KB_DATA_PATH`, права из design.md, свой провайдер `kb` только при URL, без `baseURL` для `opencode-go`.
- [x] 1.4 Парсер JSONL: `tool:<name>`, накопление `text`, событие `error` роняет прогон. API-тесты 503 для обоих агентов.

## Slice: Образ и документация

> **Type**: infra
> **Outcome**: официальный образ содержит `cursor-agent` и OpenCode 2.2.7; README и `.env.example` объясняют переключение и OpenCode Go.
> **Acceptance**: в Dockerfile есть установка `https://opencode.ai/v2/install` с `--version 2.2.7`; README содержит раздел настройки OpenCode Go с `OPENCODE_API_KEY` и моделью `opencode-go/<id>`.
> **Skills**: нет отдельного skill; следовать `AGENTS.md` (env в README и `.env.example` вместе).
> **Scope**: `Dockerfile`, `README.md`, `.env.example`
> **Allowed fallout**: комментарий стадии runtime в Dockerfile
> **Blocked**: удаление установки `cursor-agent`; публикация образа из этой задачи

- [x] 2.1 Установка OpenCode v2.2.7 пользователем `kb` в `~/.local/bin`.
- [x] 2.2 `.env.example`: `KB_NODE_AGENT`, `KB_NODE_AGENT_MODEL`, `KB_NODE_AGENT_API_URL`, `KB_NODE_AGENT_API_KEY`, `KB_NODE_AGENT_STATE_DIR`, `OPENCODE_API_KEY`.
- [x] 2.3 README: переключатель, Go, запрет своего URL для `opencode-go`, пример `docker run`.

## Slice: Текст модалки

> **Type**: webapp
> **Outcome**: пояснение модалки не называет Cursor.
> **Acceptance**: `cd web && npm test -- src/components/NodeAgentEditDialog.test.tsx`
> **Skills**: `web-frontend`, `web-frontend-tests`
> **Scope**: `web/src/components/NodeAgentEditDialog.tsx` и его тест
> **Allowed fallout**: утверждение в тесте на новый текст
> **Blocked**: новые поля, селектор агента, правки Action Bar

- [x] 3.1 Текст: «Опишите, что нужно изменить в узле. Агент отредактирует файл на сервере; ход выполнения отобразится в панели логов.»
- [x] 3.2 Vitest проверяет отсутствие «Cursor» и disabled «Запустить» при пустой инструкции.

## Gate: browser

> **Type**: browser
> **Acceptance**: выполнить `qa_plan.md`; P0 блокирует review
> **Skills**: `web-frontend-tests`
> **Scope**: W1, если поднят сервер и доступен браузер; иначе явный skip со ссылкой на Vitest
> **Allowed fallout**: точечный фикс текста модалки
> **Blocked**: новые сценарии вне qa_plan

- [x] Q.1 Выполнить автоматические проверки qa_plan и W1 либо зафиксировать skip
- [x] Q.2 Записать evidence и пробелы

Evidence: `go test ./internal/api/ ./internal/bootstrap/config/ ./internal/bootstrap/ -count=1 -race` pass; `golangci-lint` 0 issues; Vitest `NodeAgentEditDialog` 5 pass; `openspec validate` valid. W1 (браузер) пропущен: встроенный UI не пересобран, P0 закрыт Vitest.

## Gate: review

> **Type**: review
> **Acceptance**: свежий diff review; CRITICAL=0
> **Skills**: `kb-backend-golang`, `golang-errors`, `api-conventions`
> **Scope**: diff относительно `main` и артефакты change
> **Allowed fallout**: один пакет правок по принятым замечаниям
> **Blocked**: ревью тем же исполнителем; drive-by рефакторинг

- [x] R.1 Записать CRITICAL / IMPORTANT / RECOMMENDATIONS / GOOD

Review (`cursor-grok-4.6-high`): CRITICAL — allow-edit был на stem без `.md`, исправлено на `{stem}.md`. IMPORTANT — общий `opencode.json` на все джобы, исправлено отдельным job-каталогом. RECOMMENDATIONS — `APIKey` прокидывается в env; state dir внутри `KB_DATA_PATH` отклоняется. GOOD — валидация Go, дефолт Cursor, 503, маршруты, текст модалки. Повторный `go test ./internal/api/ -race` и lint после правки — pass.
