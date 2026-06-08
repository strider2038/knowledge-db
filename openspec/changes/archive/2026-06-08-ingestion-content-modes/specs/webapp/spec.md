# webapp (delta)

## ADDED Requirements

### Requirement: Content mode selector on Add page

The manual Add (ingest) form SHALL provide a content mode control with options: Auto, Verbatim, Full article fetch, Digest, Link bookmark. Default selection is Auto. Labels and short hints SHALL explain when to use each mode (Russian UI). The content mode control is the primary control for body handling. The existing `type_hint` control is a secondary storage-form hint.

#### Scenario: User selects Verbatim before submit

- **WHEN** user chooses «Как есть» and submits URL with pasted body
- **THEN** request includes `content_mode: "verbatim"`
- **AND** result points to a node whose body is preserved from paste

#### Scenario: Auto mode unchanged UX

- **WHEN** user leaves mode on Auto
- **THEN** request omits `content_mode` or sends `auto` per API contract
- **AND** behavior matches pre-change auto rules after backend implementation

#### Scenario: User selects Verbatim with Article type

- **WHEN** user chooses «Как есть» and secondary type hint «Статья»
- **THEN** UI copy explains that pasted body will be saved as an article without fetching over it
- **AND** request includes `content_mode: "verbatim"` and `type_hint: "article"`

### Requirement: Content mode selector on Telegram import tab

The Telegram import accept UI on the Add page SHALL provide the same content mode options as the manual Add form. Default selection is Auto. Accepting an item SHALL send selected `content_mode` to the import accept API.

#### Scenario: Import item accepted as bookmark

- **WHEN** user chooses «Закладка» for the current Telegram import item and accepts it
- **THEN** request includes `content_mode: "link_bookmark"`
- **AND** result shows the saved node path and keeps processing the next import item
