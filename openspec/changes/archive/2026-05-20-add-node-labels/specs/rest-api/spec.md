## Purpose

Дельта: REST API для чтения, обновления и фильтрации личных меток узлов (`labels`).

## MODIFIED Requirements

### Requirement: Список узлов с фильтрами

API MUST поддерживать GET /api/nodes с query-параметрами: path (путь темы, пустой = вся база), recursive (bool, по умолчанию false), q (подстрока поиска в title, keywords, annotation), type (article, link, note — через запятую), manual_processed (опционально: true или false — только узлы с соответствующим флагом; при отсутствии параметра возвращаются все узлы независимо от флага), labels (опционально: список меток через запятую — только узлы, содержащие **все** указанные метки; сравнение без учёта регистра после нормализации), limit, offset (пагинация). При recursive=true возвращаются узлы всего поддерева. Ответ MUST содержать nodes (массив с path, title, type, created, source_url, translations, manual_processed, labels) и total (общее количество до пагинации). Узлы без поля manual_processed в хранилище MUST трактоваться как manual_processed=false в JSON. Узлы без поля labels MUST возвращать labels как пустой массив `[]`. Переводы (slug.lang.md) не включаются как отдельные узлы. Параметр q MUST NOT искать по полю labels.

#### Сценарий: Рекурсивный список

- **WHEN** GET /api/nodes?path=programming&recursive=true
- **THEN** возвращаются узлы из programming и всех подпапок

#### Сценарий: Поиск по тексту

- **WHEN** GET /api/nodes?path=ai&recursive=true&q=go
- **THEN** возвращаются только узлы, где «go» входит в title, keywords или annotation

#### Сценарий: Фильтр по типу

- **WHEN** GET /api/nodes?path=&recursive=true&type=article,link
- **THEN** возвращаются только узлы типа article или link

#### Сценарий: Пагинация

- **WHEN** GET /api/nodes?path=ai&recursive=true&limit=20&offset=40
- **THEN** возвращаются узлы 41–60 и total для расчёта страниц

#### Сценарий: Фильтр только проверенных вручную

- **WHEN** GET /api/nodes?path=&recursive=true&manual_processed=true
- **THEN** возвращаются только узлы с manual_processed=true

#### Сценарий: Фильтр только непроверенных

- **WHEN** GET /api/nodes?path=&recursive=true&manual_processed=false
- **THEN** возвращаются только узлы без отметки или с manual_processed=false

#### Сценарий: Фильтр по одной метке

- **WHEN** GET /api/nodes?path=&recursive=true&labels=favorite
- **THEN** возвращаются только узлы, у которых в labels есть метка favorite (без учёта регистра)

#### Сценарий: Фильтр по нескольким меткам (AND)

- **WHEN** GET /api/nodes?path=&recursive=true&labels=favorite,review
- **THEN** возвращаются только узлы, содержащие и favorite, и review

#### Сценарий: Пустые сегменты labels игнорируются

- **WHEN** GET /api/nodes?labels=favorite,,review
- **THEN** API трактует запрос как labels=favorite,review

### Requirement: Обновление manual_processed

API MUST поддерживать частичное обновление метаданных узла через `PATCH /api/nodes/{path...}` и принимать одно или несколько полей из набора: `manual_processed`, `title`, `keywords`, `labels`. Для неподдерживаемых полей API MUST возвращать `400 Bad Request`. Для некорректных типов значений API MUST возвращать `400 Bad Request`.

Сервер MUST нормализовать значения перед сохранением:
- `title`: trim; пустая строка удаляет поле `title` из frontmatter.
- `keywords`: trim каждого элемента, удаление пустых значений, дедупликация с сохранением порядка.
- `manual_processed`: boolean, при `false` допускается снятие флага согласно принятому представлению optional bool.
- `labels`: массив строк; нормализация по правилам knowledge-storage (trim, dedupe case-insensitive, лимиты); пустой массив удаляет `labels` из frontmatter.

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

#### Сценарий: Обновление labels

- **WHEN** клиент отправляет PATCH с `{ "labels": ["  favorite ", "Favorite", "review"] }`
- **THEN** сервер сохраняет нормализованный список (например `["favorite", "review"]`) и возвращает узел с полем labels в JSON

#### Сценарий: Очистка labels

- **WHEN** клиент отправляет PATCH с `{ "labels": [] }`
- **THEN** ключ labels удаляется из frontmatter, в JSON labels равен `[]`

#### Сценарий: Неподдерживаемое поле

- **WHEN** клиент отправляет PATCH с неизвестным полем, например `{ "unexpected": "x" }`
- **THEN** API возвращает `400 Bad Request` и не изменяет файл узла

## ADDED Requirements

### Requirement: Подсказки существующих labels

API MUST предоставлять GET /api/label-suggestions, возвращающий JSON `{ "labels": ["...", ...] }` — уникальные метки, встречающиеся хотя бы у одного узла базы, отсортированные для UI (например, по алфавиту). Список MUST NOT включать keywords. Ответ MAY ограничиваться разумным лимитом (например, 500 записей).

#### Сценарий: Получение подсказок

- **WHEN** клиент вызывает GET /api/label-suggestions
- **THEN** возвращается массив уникальных строк labels из frontmatter узлов

### Requirement: Метаданные узла содержат labels

Ответ GET узла по пути (и любые ответы с полным телом метаданных узла, используемые веб-клиентом) MUST содержать поле labels (массив строк). Если в файле поле отсутствует, JSON MUST содержать `labels: []`.

#### Сценарий: Чтение узла без labels в файле

- **WHEN** GET узла для .md без ключа labels
- **THEN** в JSON labels равен пустому массиву
