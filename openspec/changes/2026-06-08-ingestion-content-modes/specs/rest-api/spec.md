# rest-api (delta)

## ADDED Requirements

### Requirement: Optional content_mode on ingest

`POST /api/ingest` SHALL accept optional JSON field `content_mode` with values `auto`, `verbatim`, `full_fetch`, `digest`, `link_bookmark`. Omitted field defaults to `auto`. Invalid values SHALL return HTTP 422.

#### Scenario: Ingest with verbatim override

- **WHEN** client POSTs `{"url":"...","raw_content":"...","content_mode":"verbatim"}`
- **THEN** response includes resolved mode and node body preserves raw content

#### Scenario: Invalid content_mode

- **WHEN** client POSTs `content_mode: "copy"`
- **THEN** API returns 422 with validation error

### Requirement: Ingest response exposes resolved content_mode

Successful ingest response SHALL include `content_mode` (resolved value after `auto` resolution) for debugging and UI display.

#### Scenario: Auto resolution in response

- **WHEN** client omits `content_mode`
- **THEN** response `content_mode` reflects pipeline resolution (e.g. `link_bookmark`)
