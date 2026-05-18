## MODIFIED Requirements

### Requirement: Обновление manual_processed

API MUST поддерживать частичное обновление метаданных узла через `PATCH /api/nodes/{path...}` и принимать одно или несколько полей из набора: `manual_processed`, `title`, `keywords`. Для неподдерживаемых полей API MUST возвращать `400 Bad Request`. Для некорректных типов значений API MUST возвращать `400 Bad Request`.

Сервер MUST нормализовать значения перед сохранением:
- `title`: trim; пустая строка удаляет поле `title` из frontmatter.
- `keywords`: trim каждого элемента, удаление пустых значений, дедупликация с сохранением порядка.
- `manual_processed`: boolean, при `false` допускается снятие флага согласно принятому представлению optional bool.

#### Сценарий: Установка флага manual_processed

- **WHEN** клиент отправляет PATCH с `{ "manual_processed": true }`
- **THEN** в frontmatter сохраняется `manual_processed: true`, ответ содержит обновлённый узел

#### Сценарий: Снятие флага manual_processed

- **WHEN** клиент отправляет PATCH с `{ "manual_processed": false }`
- **THEN** флаг manual_processed снимается или сохраняется как false согласно реализации, ответ содержит обновлённый узел

#### Сценарий: Обновление title

- **WHEN** клиент отправляет PATCH с `{ "title": "  New title  " }`
- **THEN** сервер сохраняет `title: "New title"` и возвращает обновлённый узел

#### Сценарий: Очистка title

- **WHEN** клиент отправляет PATCH с `{ "title": "   " }`
- **THEN** поле `title` удаляется из frontmatter

#### Сценарий: Обновление keywords с повторами и пустыми значениями

- **WHEN** клиент отправляет PATCH с `{ "keywords": ["go", "  kubernetes ", "go", ""] }`
- **THEN** сервер сохраняет `keywords: ["go", "kubernetes"]` и возвращает обновлённый узел

#### Сценарий: Неподдерживаемое поле

- **WHEN** клиент отправляет PATCH с неизвестным полем, например `{ "unexpected": "x" }`
- **THEN** API возвращает `400 Bad Request` и не изменяет файл узла
