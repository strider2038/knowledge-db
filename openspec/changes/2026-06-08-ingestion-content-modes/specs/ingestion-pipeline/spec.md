# ingestion-pipeline (delta)

## ADDED Requirements

### Requirement: Content mode axis

The ingestion pipeline SHALL resolve a **content mode** for each ingest request before LLM orchestration. Supported modes: `verbatim`, `full_fetch`, `digest`, `link_bookmark`, `auto`. When `auto`, the pipeline SHALL apply deterministic rules from source channel, presence of `raw_content`, URL-only vs body, and optional `type_hint`.

#### Scenario: Paste with body and article type hint

- **WHEN** ingest receives non-empty `raw_content`, a URL, and `type_hint=article`
- **THEN** resolved mode is `verbatim` unless caller explicitly sets `content_mode=full_fetch`
- **AND** post-LLM `ensureArticleContent` SHALL NOT replace body with URL fetch

#### Scenario: URL only without body

- **WHEN** ingest receives a URL and empty `raw_content`
- **THEN** resolved mode is `link_bookmark` (or `digest` when profile requires long-form fetch)
- **AND** pipeline SHALL generate digest content via `ensureDigestContent` on initial ingest, not only on refresh

#### Scenario: Telegram long-form message with URL

- **WHEN** classification is long-form Telegram text with embedded URL
- **THEN** default mode is `verbatim`
- **AND** LLM prompt for verbatim SHALL preserve source markdown and SHALL NOT instruct rewrite to conceptual digest

#### Scenario: Explicit content_mode override

- **WHEN** client sends `content_mode=full_fetch`
- **THEN** pipeline SHALL fetch article HTML for `type=article` even if `raw_content` is present

### Requirement: Mode-specific LLM instructions

LLM system and user prompts SHALL be selected by resolved content mode. Verbatim mode SHALL forbid digest rewrite. Digest and link_bookmark modes SHALL require non-empty digest sections per existing digest spec. Full_fetch mode SHALL allow body replacement from fetched article.

#### Scenario: Verbatim note ingest

- **WHEN** mode is `verbatim`
- **THEN** persisted `description` and translation body SHALL match source text structure (normalized whitespace only)

### Requirement: Title normalization before persist

Before writing the node, the pipeline SHALL normalize title: strip markdown link syntax, remove leading/trailing emoji used as decoration, collapse redundant whitespace.

#### Scenario: Title with markdown link

- **WHEN** LLM or source returns title `[text](url)`
- **THEN** stored title is plain `text` without link markup

### Requirement: Digest on initial ingest for link modes

For resolved modes `digest` and `link_bookmark`, `ensureDigestContent` SHALL run during initial ingest when digest is empty or placeholder, same as refresh path.

#### Scenario: New link bookmark from Telegram forward

- **WHEN** mode is `link_bookmark` and digest is empty after LLM
- **THEN** orchestrator generates digest before persist
