# ingestion-content-modes (delta)

## ADDED Requirements

### Requirement: Content mode contract

The system SHALL use `content_mode` as an operational ingest axis separate from persisted node `type` and `content_profile`. Supported values are `auto`, `verbatim`, `full_fetch`, `digest`, and `link_bookmark`.

`content_mode` SHALL be accepted in ingest requests, resolved before LLM orchestration, returned in ingest/import responses, and logged for debugging. It SHALL NOT be persisted in node frontmatter in this change.

#### Scenario: Mode is not persisted

- **WHEN** a node is created with explicit or resolved `content_mode`
- **THEN** the persisted frontmatter includes existing storage fields such as `type`, `source_kind`, and `content_profile`
- **AND** the persisted frontmatter does not include `content_mode`

### Requirement: Non-empty persisted body

Every persisted node body created or repaired by ingestion SHALL be non-empty. `link_bookmark` mode SHALL create compact semantic body for search, not an empty bookmark.

#### Scenario: Link bookmark body

- **WHEN** resolver chooses `link_bookmark`
- **THEN** the saved node body contains compact factual text derived from URL, metadata, source text, or other available source facts
- **AND** the saved node body is suitable for semantic search
