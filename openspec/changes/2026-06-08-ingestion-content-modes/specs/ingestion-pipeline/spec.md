# ingestion-pipeline (delta)

## ADDED Requirements

### Requirement: Content mode axis

The ingestion pipeline SHALL resolve a **content mode** for each ingest request before LLM orchestration. Supported modes: `verbatim`, `full_fetch`, `digest`, `link_bookmark`, `auto`. When `auto`, the pipeline SHALL apply deterministic rules from source channel, presence of original user body (`text` / internal `RawContent`), URL-only vs body, optional `type_hint`, and source classification. The resolved mode SHALL be available in `ProcessInput`, logs, and API responses, but SHALL NOT be persisted in node frontmatter.

#### Scenario: Paste with body and article type hint

- **WHEN** ingest receives non-empty request `text` / internal `RawContent`, a URL, and `type_hint=article`
- **THEN** resolved mode is `verbatim` unless caller explicitly sets `content_mode=full_fetch`
- **AND** post-LLM `ensureArticleContent` SHALL NOT replace body with URL fetch
- **AND** the persisted body is restored from original user body, not from prompt text with system/source prefixes

#### Scenario: URL only without body

- **WHEN** ingest receives a URL and no user body beyond the URL/instruction text
- **THEN** resolved mode is `full_fetch` when `type_hint=article`, `digest` when classification maps to a digest/profile source, or `link_bookmark` for minimal/unknown bookmarks
- **AND** pipeline SHALL generate non-empty body on initial ingest, not only on refresh

#### Scenario: Telegram long-form message with URL

- **WHEN** classification is long-form Telegram text with embedded URL
- **THEN** default mode is `verbatim`
- **AND** LLM prompt for verbatim SHALL preserve source markdown and SHALL NOT instruct rewrite to conceptual digest

#### Scenario: Explicit content_mode override

- **WHEN** client sends `content_mode=full_fetch`
- **THEN** pipeline SHALL fetch article HTML for `type=article` even if request `text` / internal `RawContent` is present

#### Scenario: Explicit bookmark override

- **WHEN** client sends `content_mode=link_bookmark` for a URL
- **THEN** pipeline SHALL create a `link`-oriented node with compact non-empty body based on URL/title/metadata/source text
- **AND** pipeline SHALL NOT fetch full article content solely to make a long digest

### Requirement: Mode-specific LLM instructions

LLM system and user prompts SHALL be selected by resolved content mode. Verbatim mode SHALL forbid digest rewrite. Digest mode SHALL require non-empty structured digest sections per `content_profile`. Link bookmark mode SHALL require compact non-empty semantic body suitable for search, without full-fetch rewrite. Full_fetch mode SHALL allow body replacement from fetched article.

#### Scenario: Verbatim note ingest

- **WHEN** mode is `verbatim`
- **THEN** persisted `description` and translation body SHALL match source text structure (normalized whitespace only)

### Requirement: Title normalization before persist

Before writing the node, the pipeline SHALL normalize title: strip markdown link syntax, move leading emoji/decorator symbols to the end of the title, remove duplicate decorator symbols, and collapse redundant whitespace.

#### Scenario: Title with markdown link

- **WHEN** LLM or source returns title `[text](url)`
- **THEN** stored title is plain `text` without link markup

### Requirement: Non-empty body guardrail

The pipeline SHALL enforce non-empty persisted body for every content mode through a shared `applyContentModeGuardrails` post-LLM entrypoint. The guardrail entrypoint SHALL run during initial ingest and refresh. For `verbatim`, it SHALL restore body from original user input. For `full_fetch`, it SHALL fill body from fetch/cache and fail rather than persist an empty body when full content is unavailable. For `digest`, it SHALL require a structured digest matching `content_profile`. For `link_bookmark`, it SHALL require compact semantic body from available facts.

#### Scenario: New link bookmark from Telegram forward

- **WHEN** mode is `link_bookmark` and content is empty after LLM
- **THEN** orchestrator retries or fills compact semantic body before persist

#### Scenario: New note digest from article URL

- **WHEN** mode is `digest`, `type=note`, and `content_profile=conceptual_digest`
- **THEN** empty content after LLM SHALL be retried before persist

#### Scenario: Full fetch cannot produce body

- **WHEN** mode is `full_fetch` and fetch/cache cannot provide full content
- **THEN** ingest or refresh SHALL fail without persisting an empty body

### Requirement: Refresh mode inference

Refresh-description SHALL infer content mode from persisted node fields because `content_mode` is not stored in frontmatter.

Body emptiness is a repair trigger after mode inference, not the primary mode selector. The primary selector is stored `type`, `content_profile`, and `source_url`.

#### Scenario: Refresh article

- **WHEN** a stored node has `type=article` and non-empty `source_url`
- **THEN** refresh mode is `full_fetch`
- **AND** refresh SHALL restore or update full article body from fetch/cache

#### Scenario: Refresh article without source URL

- **WHEN** a stored node has `type=article` and no `source_url`
- **THEN** refresh SHALL fail with source-url-required behavior and SHALL NOT mutate the node

#### Scenario: Refresh profile link

- **WHEN** a stored node has `type=link`, non-empty `source_url`, and `content_profile` other than empty or `link_bookmark`
- **THEN** refresh mode is `digest`
- **AND** refresh SHALL update a structured profile digest

#### Scenario: Refresh bookmark link

- **WHEN** a stored node has `type=link` and empty or `link_bookmark` `content_profile`
- **THEN** refresh mode is `link_bookmark`
- **AND** refresh SHALL keep or create compact non-empty semantic body

#### Scenario: Refresh ordinary note

- **WHEN** a stored node has `type=note` without a digest `content_profile`
- **THEN** refresh mode is `verbatim`
- **AND** refresh SHALL NOT rewrite the note body into a digest

#### Scenario: Refresh note digest

- **WHEN** a stored node has `type=note`, `content_profile=conceptual_digest` or `content_profile=brief_digest`, and non-empty `source_url`
- **THEN** refresh mode is `digest`
- **AND** refresh SHALL update a structured note digest

#### Scenario: Repair empty body

- **WHEN** a stored node has empty body and non-empty `source_url`
- **THEN** refresh SHALL infer mode from stored fields and regenerate non-empty body for that mode
- **AND** refresh SHALL fail without mutation when there is no `source_url` and no original text/facts to build body from

## MODIFIED Requirements

### Requirement: Сохранение полной статьи по явному намерению

Pipeline SHALL preserve the existing full-article behavior only when resolved content mode is `full_fetch`. `TypeHint=article` alone no longer means body replacement by fetch when the resolver chooses `verbatim` because request text contains substantial user-provided body.

#### Scenario: Article type hint with pasted body

- **WHEN** request has `type_hint=article`, non-empty pasted body, and no explicit `content_mode=full_fetch`
- **THEN** resolved mode is `verbatim`
- **AND** final node may have `type=article`, but body SHALL remain the pasted body

### Requirement: Type-aware refresh внешних узлов

RefreshDescription SHALL infer content mode from stored `type`, `content_profile`, and `source_url`, then apply `applyContentModeGuardrails`. This replaces refresh behavior that always assumes digest generation for every external node. Ordinary notes without digest profile are refreshed in `verbatim` mode and SHALL NOT have their body rewritten.
