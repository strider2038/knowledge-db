## Purpose

Интеграция git-операций в UI: кнопка «Коммит» в верхней панели для фиксации изменений с автогенерацией commit message через LLM.

## Requirements

### Requirement: API проверки git-статуса

REST API MUST предоставлять эндпоинт `GET /api/git/status`, возвращающий `{ "has_changes": bool, "changed_files": number }`. При отключённом git (KB_GIT_DISABLED=true) MUST возвращаться 503 с сообщением "git is disabled".

#### Сценарий: Есть незакоммиченные изменения

- **WHEN** `GET /api/git/status` и в git есть modified/untracked/deleted файлы
- **THEN** возвращается `{ "has_changes": true, "changed_files": N }` где N — количество изменённых файлов

#### Сценарий: Нет изменений

- **WHEN** `GET /api/git/status` и рабочий каталог чист
- **THEN** возвращается `{ "has_changes": false, "changed_files": 0 }`

#### Сценарий: Git отключён

- **WHEN** `GET /api/git/status` и KB_GIT_DISABLED=true
- **THEN** возвращается 503

### Requirement: API коммита с LLM-генерацией сообщения

REST API MUST предоставлять эндпоинт `POST /api/git/commit` для коммита всех изменений. Тело запроса MAY содержать `{ "message"?: string }` — ручное сообщение (при отсутствии генерируется через LLM). При успехе MUST возвращаться `{ "message": "commit message", "committed": true }`. API MUST выполнять `git add -A`, `git commit -m <message>`, `git push`. Генерация через LLM MUST использовать OpenAI Responses API: на вход — `git diff --stat`, instructions — сгенерировать conventional commit message. При недоступности LLM MUST использоваться fallback-сообщение `chore: manual commit via UI`. При отключённом git — 503.

#### Сценарий: Коммит с автогенерацией сообщения

- **WHEN** `POST /api/git/commit` без тела (или без поля message) и есть изменения
- **THEN** сервер получает diff, отправляет в LLM, получает commit message, выполняет git add -A + commit + push, возвращает сгенерированное сообщение

#### Сценарий: Коммит с ручным сообщением

- **WHEN** `POST /api/git/commit` с `{ "message": "fix: update docs" }`
- **THEN** выполняется коммит с указанным сообщением без вызова LLM

#### Сценарий: Нет изменений для коммита

- **WHEN** `POST /api/git/commit` и нет незакоммиченных изменений
- **THEN** возвращается 400 или 200 с `{ "committed": false, "message": "no changes to commit" }`

#### Сценарий: LLM недоступна

- **WHEN** `POST /api/git/commit` и LLM-сервис не сконфигурирован или недоступен
- **THEN** используется fallback-сообщение `chore: manual commit via UI`, коммит выполняется

#### Сценарий: Git отключён

- **WHEN** `POST /api/git/commit` и KB_GIT_DISABLED=true
- **THEN** возвращается 503

### Requirement: Кнопка «Коммит» в Navbar

UI MUST отображать кнопку «Коммит» в Navbar, если git включён и есть незакоммиченные изменения. Кнопка MUST показывать количество изменённых файлов. UI MUST периодически (каждые 30 секунд) проверять git-статус через `GET /api/git/status`. При нажатии MUST вызываться `POST /api/git/commit`. Во время выполнения кнопка MUST отображать спиннер и быть disabled. При успехе — кнопка скрывается (или обновляется статус). При ошибке — отображается уведомление.

#### Сценарий: Кнопка видна при наличии изменений

- **WHEN** `GET /api/git/status` возвращает `has_changes: true`
- **THEN** в Navbar отображается кнопка «Коммит» с количеством изменённых файлов

#### Сценарий: Кнопка скрыта при отсутствии изменений

- **WHEN** `GET /api/git/status` возвращает `has_changes: false`
- **THEN** кнопка «Коммит» не отображается

#### Сценарий: Успешный коммит

- **WHEN** пользователь нажимает «Коммит» и API возвращает успех
- **THEN** отображается краткое уведомление с commit message, кнопка скрывается

#### Сценарий: Ошибка при коммите

- **WHEN** пользователь нажимает «Коммит» и API возвращает ошибку
- **THEN** отображается уведомление с текстом ошибки, кнопка остаётся видимой

#### Сценарий: Git отключён на клиенте

- **WHEN** `GET /api/git/status` возвращает 503
- **THEN** кнопка «Коммит» не отображается
