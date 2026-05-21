## Purpose

Дельта: дедупликация при ingestion (update vs create).

## MODIFIED Requirements

### Requirement: Создание узла при ingestion

При успешной обработке система ДОЛЖНА (SHALL) сохранить результат в базе знаний: либо **обновить** существующий узел, либо **создать** новый — по правилам node-identity (lookup по `node_id` если передан, иначе по нормализованному `source_url`, иначе create). При create: директория с файлом `{slug}.md`, frontmatter с метаданными включая новый `id` (UUID v7) и markdown-контент. При update: тот же файл узла (по найденному path), `id` MUST NOT изменяться. После сохранения MUST выполнить git add + commit. Frontmatter MUST содержать поля `title` и `aliases` (если LLM вернул title). Frontmatter MUST содержать `source_author` при наличии в результате create_node.

#### Сценарий: Создание узла для статьи

- **WHEN** pipeline обработал URL статьи, для которого нет узла с таким source_url
- **THEN** создаётся узел с type=article, id, source_url, source_date, source_author (если извлечён), полным контентом и git commit

#### Сценарий: Обновление узла для статьи при повторном URL

- **WHEN** pipeline обработал URL статьи, для которого уже есть узел с тем же нормализованным source_url
- **THEN** обновляется существующий файл узла, id сохраняется, новый файл не создаётся, выполняется git commit

#### Сценарий: Создание узла для заметки

- **WHEN** pipeline обработал текст без URL
- **THEN** создаётся узел с type=note, новым id, контентом из текста и git commit

#### Сценарий: Создание узла для ссылки

- **WHEN** pipeline обработал URL на ресурс/сервис без существующего узла с таким source_url
- **THEN** создаётся узел с type=link, id, source_url, аннотацией и git commit

#### Сценарий: Создание узла с title

- **WHEN** LLM вернул непустой title в create_node
- **THEN** frontmatter создаваемого или обновляемого узла содержит поля `title` и `aliases: [<title>]`

#### Сценарий: Создание узла с source_author

- **WHEN** create_node вернул source_author (от LLM или подставлен из FetchResult)
- **THEN** frontmatter узла содержит поле `source_author`

#### Сценарий: Генерация title при отсутствии в источнике

- **WHEN** контент не содержит явного заголовка (заметка, пересланное сообщение без title)
- **THEN** LLM MUST сгенерировать осмысленный title на основе содержимого и передать его в create_node

#### Сценарий: Тема не существует

- **WHEN** LLM указал тему, которой нет в базе, и выполняется create (не update)
- **THEN** система создаёт промежуточные директории для новой темы

#### Сценарий: Update не меняет theme/slug при дедупе по URL

- **WHEN** ingestion выполняет update существующего узла по source_url
- **THEN** path и slug файла MUST оставаться прежними (обновляются только метаданные и body), если явно не запрошено иное
