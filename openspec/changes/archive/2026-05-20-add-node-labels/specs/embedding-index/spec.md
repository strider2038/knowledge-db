## Purpose

Дельта: исключение личных меток (`labels`) из embedding-индекса.

## MODIFIED Requirements

### Requirement: Embedding-индекс для нод

Система ДОЛЖНА (SHALL) генерировать и хранить embedding для каждой ноды. Текст для embedding SHALL формироваться: `title + " " + annotation + " " + keywords` — для всех типов; для `note` — дополнительно `+ " " + body`; для `link` — дополнительно `+ " " + body`, если узел содержит `content_profile` и непустое markdown-тело. Поле `labels` MUST NOT входить в текст embedding и MUST NOT входить в searchable text для смыслового/keyword поиска по контенту. Embedding MUST храниться как BLOB (little-endian float32 array) в таблице embeddings. Система MUST хранить content_hash (hash от title+annotation+keywords+type+source_kind+content_profile; **без** labels) и body_hash (hash от body) для определения изменений. При индексации ноды система MUST обновлять searchable text записи для keyword/FTS поиска. Searchable text MUST включать `source_kind`, `content_profile` и body для `note` и профильных `link` узлов. Система MUST создавать chunks для `article`, а также для `note` и `link` узлов с digest body, если body достаточно длинное для chunking.

#### Scenario: Индексация новой ноды

- **WHEN** SyncWorker обрабатывает ноду, отсутствующую в indexed_nodes
- **THEN** генерируется node embedding, создаётся запись в indexed_nodes и embeddings, а searchable text ноды доступен для keyword/FTS поиска

#### Scenario: Обновление при изменении метаданных

- **WHEN** нода существует в indexed_nodes, но content_hash отличается
- **THEN** node embedding и searchable text ноды пересчитываются, indexed_nodes обновляется

#### Scenario: Обновление при изменении body статьи

- **WHEN** нода типа article существует в indexed_nodes, но body_hash отличается
- **THEN** старые чанки и их searchable text удаляются, генерируются новые чанки и их embeddings

#### Scenario: Индексация repository profile body

- **WHEN** нода `type=link` содержит `content_profile=repository_profile` и markdown-тело
- **THEN** body включается в node embedding, searchable text и chunk index при достаточном размере

#### Scenario: Индексация conceptual digest note

- **WHEN** нода `type=note` содержит `content_profile=conceptual_digest` и markdown-тело
- **THEN** body включается в node embedding, searchable text и chunk index при достаточном размере

#### Scenario: Старый link без digest

- **WHEN** нода `type=link` не содержит `content_profile` и имеет пустое тело
- **THEN** индексирование продолжает использовать title, annotation и keywords без ошибки

#### Scenario: Изменение только labels

- **WHEN** у узла изменилось только поле labels во frontmatter
- **THEN** content_hash не меняется и переиндексация embedding не выполняется
