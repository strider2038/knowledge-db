# telegram-bot (delta)

## ADDED Requirements

### Requirement: Telegram ingest uses content mode resolver

Telegram live ingest SHALL pass incoming message text, delivery `source_url`, and `source_author` through the same content mode resolver as API ingest. Telegram bot does not need inline mode-selection buttons in this change; auto resolution is sufficient for the required behavior.

#### Scenario: Long-form Telegram message with URL

- **WHEN** Telegram bot receives long-form text with an embedded URL
- **THEN** resolved content mode is `verbatim` by default
- **AND** persisted body preserves the message text instead of rewriting it into a digest

#### Scenario: URL-only Telegram forward

- **WHEN** Telegram bot receives a URL-only message or forward
- **THEN** resolved content mode is `full_fetch`, `digest`, or `link_bookmark` according to the resolver table
- **AND** persisted body is non-empty for semantic search

#### Scenario: Telegram bookmark fallback

- **WHEN** resolver chooses `link_bookmark`
- **THEN** the bot-created node contains compact semantic body based on available URL/meta/source facts
