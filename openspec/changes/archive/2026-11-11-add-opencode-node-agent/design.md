## Context

Нормализация и agent-edit вызывают `RunCursorAgent` (`cursor-agent --print --output-format stream-json --force`) с `cmd.Dir = KB_DATA_PATH`. Доступность бинаря проверяется type assert на конкретный раннер. Образ ставит только Cursor. Конфиг приложения не знает про агента узла.

OpenCode v2 запускается как `opencode run`. Провайдер подписки Go — `opencode-go`, ключ читается из `OPENCODE_API_KEY`. Свой `baseURL` для этого провайдера перекрывает каталог и ломает авторизацию. Версия CLI, зафиксированная этим изменением: `2.2.7` (`@opencode/cli`).

## Goals / Non-Goals

**Goals:**

- Один переключатель `KB_NODE_AGENT` на обе джобы. По умолчанию остаётся Cursor.
- OpenCode v2 в том же job-потоке: промпт, логи, sync после успеха.
- Подписка OpenCode Go через модель `opencode-go/<id>` и `OPENCODE_API_KEY`.
- Свой OpenAI-совместимый endpoint отдельной парой URL/ключ.
- Состояние OpenCode вне git-репозитория базы. Запись агента ограничена файлом узла.
- Оба CLI в официальном образе. Настройка описана в README и `.env.example`.

**Non-Goals:**

- Выбор агента в UI и отдельный агент для нормализации и правки.
- Удаление Cursor и смена HTTP-маршрутов.
- Живой вызов модели в тестах CI.
- Подхват личного `~/.config/opencode` и интерактивный `opencode auth login` внутри джобы.

## Decisions and Alternatives

1. Общий раннер с `Run` и `Available`. Нормализация и agent-edit только собирают промпт. Type assert на `*cursorNodeNormalizer` / `*cursorNodeAgentEditor` удаляется. Альтернатива — два независимых CLI-вызова: дублирует проверку PATH и парсинг.

2. Валидация конфигурации на старте `kb serve` (`Config.ValidateNodeAgent` рядом с остальными проверками в `bootstrap.Run`):
   - `KB_NODE_AGENT` пустой или `cursor` — успех, остальные `KB_NODE_AGENT_*` не обязательны;
   - `opencode` — `KB_NODE_AGENT_MODEL` обязателен;
   - модель `opencode-go/*` — обязателен непустой `OPENCODE_API_KEY`, `KB_NODE_AGENT_API_URL` запрещён;
   - задан `KB_NODE_AGENT_API_URL` — обязателен `KB_NODE_AGENT_API_KEY`, модель не может быть `opencode-go/*`;
   - иное значение `KB_NODE_AGENT` — ошибка старта.
   Отсутствие бинаря не валится на старте: джоба возвращает 503. Для `cursor` текст по-прежнему содержит `cursor-agent not found in PATH`. Для `opencode` текст называет `opencode`.

3. Команда OpenCode: `opencode run --standalone --format json --archived --dir <KB_DATA_PATH> --model <model> <prompt>`, рабочий каталог тот же. `--auto` не используется. Промпт остаётся аргументом, как у Cursor.

4. Дочернему процессу задаются свои `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME` под каталогом состояния. По умолчанию это `$UserCacheDir/knowledge-db/opencode`, иначе временный каталог. Переопределение — `KB_NODE_AGENT_STATE_DIR`. В `$XDG_CONFIG_HOME/opencode/opencode.json` kb пишет только права, `snapshots: false`, `update: "disable"`, `formatter: false` и, при своём endpoint, провайдер `kb`. Файл не попадает в `KB_DATA_PATH`.

5. Права, последнее совпадение побеждает: allow `read`/`glob`/`grep`; deny `edit` на `*`, затем allow `edit` только на относительный path узла; deny `shell`, `webfetch`, `websearch`, `subagent`, `question`, `external_directory`.

6. Свой endpoint: в конфиг добавляется провайдер `kb` (`package`: `@opencode/ai/providers/openai-compatible`, `env`: `KB_NODE_AGENT_API_KEY`, `settings.baseURL`: URL). `--model` становится `kb/<id>`, а `modelID` модели равен `KB_NODE_AGENT_MODEL`. Для `opencode-go/*` блок `providers` не пишется, `--model` равен значению переменной, `OPENCODE_API_KEY` наследуется из окружения сервера.

7. Логи OpenCode: `text` собирается в текст ответа, `tool_use` пишется как `tool:<name>` без тела результата, `error` помечает прогон неуспешным. Неизвестный JSON остаётся короткой строкой. Парсер Cursor не меняется.

8. Образ: существующая установка `cursor-agent` плюс `curl -fsSL https://opencode.ai/v2/install | bash -s -- --version 2.2.7` от пользователя `kb`. PATH уже включает `~/.local/bin`.

9. UI: одна фраза в `NodeAgentEditDialog` без слова Cursor. Селектора нет.

## Risks / Trade-offs

- Форма JSONL привязана к OpenCode 2.2.7. Смена версии в образе без обновления парсера даст сырые строки в логе, а событие `error` может перестать распознаваться. Версия закреплена, парсер покрыт фикстурой.
- Запрет shell не даёт модели править файл через терминал. Если edit-инструмент не справляется, джоба честно падает, а не коммитит через git.
- Длинный промпт в argv упирается в лимит ОС так же, как текущий вызов Cursor.
- Изолированный XDG не видит ключ, сохранённый интерактивным `opencode auth login`. Для сервера это сознательно: секрет задаётся `OPENCODE_API_KEY`.

## Migration / Rollout / Rollback

Существующие установки без новой переменной остаются на Cursor. Откат — убрать `KB_NODE_AGENT` или вернуть `cursor`. Образ после выката содержит оба бинаря; откат бинаря не требуется, чтобы вернуть прежнее поведение. Схема БД и файлы базы не мигрируют.

## Open Questions

Нет. Блокирующих решений не осталось.
