# ingest-type-hint (delta)

## MODIFIED Requirements

### Requirement: Подсказка типа в IngestRequest

`TypeHint` remains a storage-form hint for the final node `type` (`article`, `link`, `note`). It SHALL NOT decide how the node body is obtained when `content_mode` is available. Resolved `content_mode` controls body handling and may prevent fetch replacement even when `TypeHint=article`.

#### Сценарий: TypeHint = article with verbatim mode

- **WHEN** IngestText receives `TypeHint = "article"` and resolved `content_mode = "verbatim"`
- **THEN** the final node type is `article`
- **AND** the persisted body is restored from original user text instead of fetched URL content

### Requirement: Передача type_hint через API

Unknown `type_hint` values continue to be treated as `auto` for compatibility. This compatibility rule does not apply to `content_mode`; invalid non-empty `content_mode` values are rejected by the REST API.
