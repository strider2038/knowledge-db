## Purpose

Дельта: перенос sidecar аннотаций при move узла.

## MODIFIED Requirements

### Requirement: API перемещения узла

REST API MUST предоставлять эндпоинт `POST /api/nodes/{path}/move` для перемещения узла. Тело запроса MUST содержать `target_path` (полный целевой путь: `{тема/.../тема}/{slug}`). При успехе MUST возвращаться обновлённый объект узла (как GET) со статусом 200; объект MUST содержать тот же `id`, что до перемещения, и новый `path`. Если узел не найден — 404. Если целевой путь занят — 409 с сообщением о конфликте. Перемещение MUST переносить главный `.md` файл, директорию вложений и файл `annotations.yaml` (если существует). Промежуточные директории целевого пути MUST создаваться рекурсивно при необходимости. Поле `id` во frontmatter MUST NOT изменяться.

#### Сценарий: Успешное перемещение в другую тему

- **WHEN** `POST /api/nodes/old-topic/my-node/move` с `{ "target_path": "new-topic/my-node" }`
- **THEN** файл перемещается, возвращается объект с `path`=`new-topic/my-node` и неизменным `id`

#### Сценарий: Перемещение с изменением slug

- **WHEN** `POST /api/nodes/topic/old-name/move` с `{ "target_path": "topic/new-name" }`
- **THEN** файл переименован, `id` в frontmatter и в JSON-ответе совпадает с id до move

#### Сценарий: Целевой путь занят

- **WHEN** `POST /api/nodes/topic/my-node/move` с `{ "target_path": "other/my-node" }` и в `other/` уже существует `my-node.md`
- **THEN** возвращается 409 с сообщением "target path already exists"

#### Сценарий: Узел не найден

- **WHEN** `POST /api/nodes/nonexistent/move` с `{ "target_path": "target/node" }`
- **THEN** возвращается 404

#### Сценарий: Рекурсивное создание промежуточных директорий

- **WHEN** `POST /api/nodes/topic/node/move` с `{ "target_path": "new/deep/path/node" }` и директории `new/`, `new/deep/`, `new/deep/path/` не существуют
- **THEN** создаются все промежуточные директории, узел перемещается, `id` сохраняется

#### Сценарий: Пустой target_path

- **WHEN** `POST /api/nodes/topic/node/move` с `{ "target_path": "" }` или без поля target_path
- **THEN** возвращается 400 с сообщением "target_path is required"

#### Сценарий: Некорректный target_path

- **WHEN** `POST /api/nodes/topic/node/move` с `{ "target_path": "../etc/passwd" }` (path traversal)
- **THEN** возвращается 400 с сообщением "invalid target_path"

#### Сценарий: Перемещение с annotations.yaml

- **WHEN** у узла существует `topic/node/annotations.yaml` и выполняется успешный move
- **THEN** файл оказывается в `{target}/annotations.yaml` с тем же содержимым
