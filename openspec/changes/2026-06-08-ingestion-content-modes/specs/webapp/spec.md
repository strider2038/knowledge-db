# webapp (delta)

## ADDED Requirements

### Requirement: Content mode selector on Add page

The Add (ingest) form SHALL provide a content mode control with options: Auto, Verbatim, Full article fetch, Digest, Link bookmark. Default selection is Auto. Labels and short hints SHALL explain when to use each mode (Russian UI).

#### Scenario: User selects Verbatim before submit

- **WHEN** user chooses «Дословно» and submits URL with pasted body
- **THEN** request includes `content_mode: "verbatim"`
- **AND** preview/result shows body unchanged from paste

#### Scenario: Auto mode unchanged UX

- **WHEN** user leaves mode on Auto
- **THEN** request omits `content_mode` or sends `auto` per API contract
- **AND** behavior matches pre-change auto rules after backend implementation
