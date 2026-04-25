## Purpose

Дельта к REST API: поле `manual_processed` в метаданных узла, фильтр списка и обновление флага.

## Requirements

## MODIFIED Requirements

### Requirement: Список узлов с фильтрами

API MUST поддерживать GET /api/nodes с query-параметрами: path (путь темы, пустой = вся база), recursive (bool, по умолчанию false), q (подстрока поиска в title, keywords, annotation), type (article, link, note — через запятую), manual_processed (опционально: true или false — только узлы с соответствующим флагом; при отсутствии параметра возвращаются все узлы независимо от флага), limit, offset (пагинация). При recursive=true возвращаются узлы всего поддерева. Ответ MUST содержать nodes (массив с path, title, type, created, source_url, translations, manual_processed) и total (общее количество до пагинации). Узлы без поля manual_processed в хранилище MUST трактоваться как manual_processed=false в JSON. Переводы (slug.lang.md) не включаются как отдельные узлы.

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

## ADDED Requirements

### Requirement: Метаданные узла содержат manual_processed

Ответ GET узла по пути (и любые ответы с полным телом метаданных узла, используемые веб-клиентом) MUST содержать boolean поле manual_processed (false, если в файле поле отсутствует).

#### Сценарий: Чтение узла без поля в файле

- **WHEN** GET узла для .md без ключа manual_processed
- **THEN** в JSON manual_processed равен false

### Requirement: Обновление manual_processed

API MUST позволять установить или снять флаг manual_processed при сохранении метаданных узла тем же способом, как обновляются прочие редактируемые поля frontmatter (один запрос на сохранение метаданных узла). Некорректный тип значения MUST приводить к 400.

#### Сценарий: Установка флага

- **WHEN** клиент отправляет сохранение метаданных с manual_processed=true
- **THEN** в файле узла в frontmatter записывается manual_processed: true (или эквивалентный YAML), updated обновляется по правилам Store

#### Сценарий: Снятие флага

- **WHEN** клиент отправляет manual_processed=false
- **THEN** в frontmatter флаг снят или записан как false согласно принятому в реализации представлению опциональных булевых полей
