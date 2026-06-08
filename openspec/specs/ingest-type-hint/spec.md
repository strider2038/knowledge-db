## Purpose

Подсказка типа контента при ingestion — позволяет пользователю указать тип (article, link, note) для выбора оркестратором при создании узла. При отсутствии подсказки (auto) оркестратор определяет тип автоматически по содержимому.

## Requirements

### Requirement: Подсказка типа в IngestRequest

`TypeHint` remains a storage-form hint for the final node `type` (`article`, `link`, `note`). It SHALL NOT decide how the node body is obtained when `content_mode` is available. Resolved `content_mode` controls body handling and may prevent fetch replacement even when `TypeHint=article`. Допустимые значения: пустая строка или "auto" (автоопределение), "article", "link", "note". При TypeHint = "" или "auto" оркестратор MUST определять тип по содержимому текста. При TypeHint = "article", "link" или "note" оркестратор MUST использовать указанный тип при вызове create_node и MUST применять его к итоговому результату ingestion, даже если LLM вернул другой `type` в аргументах create_node.

#### Сценарий: TypeHint = auto

- **WHEN** IngestText вызывается с TypeHint = "" или "auto"
- **THEN** оркестратор определяет тип (article/link/note) по тексту и контексту

#### Сценарий: TypeHint = article

- **WHEN** IngestText вызывается с TypeHint = "article"
- **THEN** оркестратор создаёт узел с type=article, используя подсказку пользователя

#### Сценарий: TypeHint = link

- **WHEN** IngestText вызывается с TypeHint = "link"
- **THEN** оркестратор создаёт узел с type=link

#### Сценарий: TypeHint = note

- **WHEN** IngestText вызывается с TypeHint = "note"
- **THEN** оркестратор создаёт узел с type=note

#### Сценарий: Конфликт TypeHint и ответа LLM

- **WHEN** IngestText вызывается с TypeHint = "article", а LLM в create_node возвращает `type=note`
- **THEN** итоговый узел создаётся с `type=article`

#### Сценарий: TypeHint = article with verbatim mode

- **WHEN** IngestText receives `TypeHint = "article"` and resolved `content_mode = "verbatim"`
- **THEN** the final node type is `article`
- **AND** the persisted body is restored from original user text instead of fetched URL content

### Requirement: Передача type_hint через API

API ДОЛЖЕН (SHALL) принимать опциональное поле type_hint в теле POST /api/ingest. Допустимые значения: "auto", "article", "link", "note". При отсутствии или неизвестном значении MUST трактовать как "auto". Unknown `type_hint` values SHALL continue to be treated as `auto` for compatibility. This compatibility rule does not apply to `content_mode`; invalid non-empty `content_mode` values SHALL be rejected by the REST API with HTTP 400.

#### Сценарий: Отправка с type_hint

- **WHEN** POST /api/ingest с телом { "text": "...", "type_hint": "article" }
- **THEN** текст и type_hint передаются в Ingester, оркестратор использует подсказку

#### Сценарий: Отправка без type_hint

- **WHEN** POST /api/ingest с телом { "text": "..." }
- **THEN** type_hint трактуется как "auto", оркестратор определяет тип автоматически

#### Scenario: Unknown type_hint remains compatible

- **WHEN** client POSTs `{"text":"...","type_hint":"unknown-value"}`
- **THEN** API treats `type_hint` as `auto`
- **AND** ingest proceeds without HTTP 400 for the unknown type hint

#### Scenario: Invalid content_mode is rejected

- **WHEN** client POSTs `{"text":"...","content_mode":"copy"}`
- **THEN** API returns HTTP 400 with `invalid content_mode`
