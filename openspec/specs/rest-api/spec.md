## Purpose

REST API для CRUD операций с узлами базы знаний, полнотекстового и ключевого поиска. В scaffold — минимальный набор эндпоинтов.

## Requirements

### Requirement: Конфигурация через KB_DATA_PATH

API ДОЛЖЕН (SHALL) использовать путь к базе из переменной окружения KB_DATA_PATH.

#### Сценарий: Запуск без KB_DATA_PATH

- **WHEN** kb-server запущен без KB_DATA_PATH
- **THEN** сервер возвращает ошибку конфигурации или не стартует

### Requirement: CRUD узлов

API MUST предоставлять эндпоинты для создания, чтения, обновления и удаления узлов (в scaffold — каркас/заглушки).

#### Сценарий: Получение узла по пути

- **WHEN** GET /api/nodes/{path}
- **THEN** возвращается узел или 404

#### Сценарий: Получение дерева тем

- **WHEN** GET /api/tree
- **THEN** возвращается иерархическое дерево тем и подтем

### Requirement: Ingestion

API MUST предоставлять эндпоинт POST /api/ingest для приёма текста и передачи в ingestion pipeline.

#### Сценарий: Отправка текста

- **WHEN** POST /api/ingest с телом { "text": "..." }
- **THEN** текст передаётся в Ingester, возвращается результат

### Requirement: Список узлов с фильтрами

API MUST поддерживать GET /api/nodes с query-параметрами: path (путь темы, пустой = вся база), recursive (bool, по умолчанию false), q (подстрока поиска в title, keywords, annotation), type (article, link, note — через запятую), limit, offset (пагинация). При recursive=true возвращаются узлы всего поддерева. Ответ MUST содержать nodes (массив с path, title, type, created, source_url, translations) и total (общее количество до пагинации). Переводы (slug.lang.md) не включаются как отдельные узлы.

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

### Requirement: Поиск

Поиск по ключевым словам и подстроке в title/annotation MUST осуществляться через GET /api/nodes с параметром q. Полнотекстовый поиск по content — опционально (в scaffold — каркас).

#### Сценарий: Поиск по запросу

- **WHEN** GET /api/nodes?q=... (с path и recursive при необходимости)
- **THEN** возвращается список подходящих узлов с метаданными (nodes, total)

### Requirement: Раздача статики UI

API ДОЛЖЕН (SHALL) раздавать встроенную статику веб-интерфейса (embedded из internal/ui).

#### Сценарий: Запрос корня

- **WHEN** GET / (или /index.html)
- **THEN** возвращается index.html веб-приложения
