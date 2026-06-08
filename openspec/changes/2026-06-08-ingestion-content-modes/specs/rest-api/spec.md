# rest-api (delta)

## ADDED Requirements

### Requirement: Optional content_mode on ingest

`POST /api/ingest` SHALL keep the existing request fields `text` (required), `source_url` (optional), `source_author` (optional), and `type_hint` (optional). It SHALL also accept optional JSON field `content_mode` with values `auto`, `verbatim`, `full_fetch`, `digest`, `link_bookmark`. Omitted or empty `content_mode` defaults to `auto`. Invalid non-empty `content_mode` values SHALL return HTTP 400 with error `invalid content_mode`. Unknown `type_hint` remains legacy-compatible and is treated as `auto`.

#### Scenario: Ingest with verbatim override

- **WHEN** client POSTs `{"text":"...","source_url":"https://example.com","content_mode":"verbatim"}`
- **THEN** response includes resolved mode and node body preserves the submitted `text`

#### Scenario: Invalid content_mode

- **WHEN** client POSTs `content_mode: "copy"`
- **THEN** API returns HTTP 400 with `{"error":"invalid content_mode"}`

### Requirement: Ingest response exposes resolved content_mode

Successful ingest response SHALL be an envelope with `node` and `content_mode`. The `content_mode` value is the resolved value after `auto` resolution and is for debugging/UI display. It SHALL NOT be written to node frontmatter.

```json
{
  "node": {
    "path": "topic/example"
  },
  "content_mode": "link_bookmark"
}
```

#### Scenario: Auto resolution in response

- **WHEN** client omits `content_mode`
- **THEN** response `content_mode` reflects pipeline resolution (e.g. `link_bookmark`)

### Requirement: Optional content_mode on Telegram import accept

`POST /api/import/telegram/session/{id}/accept` SHALL accept optional JSON field `content_mode` with the same enum and validation policy as `POST /api/ingest`. It SHALL keep existing `type_hint` support.

#### Scenario: Import accept with verbatim override

- **WHEN** client POSTs `{"content_mode":"verbatim"}` to accept the current import item
- **THEN** the import item is ingested in verbatim mode
- **AND** response includes `node`, `next_item`, and resolved `content_mode`

#### Scenario: Import accept with invalid content_mode

- **WHEN** client POSTs `{"content_mode":"copy"}` to accept the current import item
- **THEN** API returns HTTP 400 with `{"error":"invalid content_mode"}`

## MODIFIED Requirements

### Requirement: Ingestion

`POST /api/ingest` SHALL return the envelope `{ "node": <node>, "content_mode": "<resolved>" }` instead of a bare node when this change is implemented. The request field `type_hint` remains a storage-form hint; body handling is controlled by resolved `content_mode`.

### Requirement: Обновление описания узла из источника

`POST /api/nodes/{path}/refresh-description` SHALL keep requiring `source_url` for source-based refresh and SHALL return the updated node object. Its internal behavior SHALL use refresh mode inference from the ingestion-pipeline spec instead of always treating refresh as digest generation. For ordinary notes without digest profile, refresh SHALL NOT rewrite the markdown body into a digest.
