## 1. Storage Model

- [ ] 1.1 Add `source_kind` and `content_profile` fields to kb frontmatter parsing/serialization models.
- [ ] 1.2 Add validation for allowed `source_kind` values while keeping missing fields valid.
- [ ] 1.3 Add validation for allowed `content_profile` values while keeping missing fields valid.
- [ ] 1.4 Add unit tests for old link nodes, repository profile links, conceptual digest notes, and invalid profile values.

## 2. Source Classification

- [ ] 2.1 Define Go constants/types for supported source kinds and content profiles.
- [ ] 2.2 Implement source classification helper for repository, documentation, product/service, online tool, directory/catalog, learning resource, article, news, social post, and unknown.
- [ ] 2.3 Detect repository URLs for GitHub and prepare extension points for GitLab/Codeberg.
- [ ] 2.4 Add tests for representative URLs from the knowledge base examples.

## 3. Fetching Source Material

- [ ] 3.1 Extend repository metadata fetching to retrieve README content or README preview for digest generation.
- [ ] 3.2 Reuse existing content fetcher for article/news/documentation preview when a digest body is needed.
- [ ] 3.3 Ensure profile generation does not store full README or full article content unless the selected flow is `type=article`.
- [ ] 3.4 Add tests for README/content availability and fallback behavior.

## 4. LLM Orchestration

- [ ] 4.1 Extend LLM tool schema for `create_node` with `source_kind` and `content_profile`.
- [ ] 4.2 Update ingestion prompts to classify source kind and select `type` based on local storage form.
- [ ] 4.3 Add digest generation instructions for repository, product, documentation, online tool, directory, learning resource, conceptual, and brief profiles.
- [ ] 4.4 Add prompt rules that exclude installation, quick start, commands, long usage examples, API reference, badges, changelog, license, sponsor, contributing, and non-conceptual benchmark tables.
- [ ] 4.5 Add orchestrator tests for repository profile, long article conceptual digest, news brief digest, and explicit full article copy.

## 5. Node Creation

- [ ] 5.1 Persist `source_kind` and `content_profile` in frontmatter when returned by the orchestrator.
- [ ] 5.2 Persist generated digest body in `content` for profile `link` and `note` nodes.
- [ ] 5.3 Preserve existing ordinary link bookmark behavior with optional empty body.
- [ ] 5.4 Add ingestion pipeline tests for `type=link/repository_profile`, `type=note/conceptual_digest`, `type=note/brief_digest`, and `type=article` full copy.

## 6. Indexing And RAG Context

- [ ] 6.1 Include `source_kind` and `content_profile` in content hash computation for indexed nodes.
- [ ] 6.2 Include body in node embedding text for `type=link` nodes with non-empty `content_profile`.
- [ ] 6.3 Include body in searchable text for `type=link` profile nodes and `type=note` digest nodes.
- [ ] 6.4 Create chunks for digest bodies when `type=link` or `type=note` body is long enough.
- [ ] 6.5 Add index/retrieval tests proving digest-only terms are found by keyword and vector/chunk retrieval paths.

## 7. Existing Node Refresh API And UI

- [ ] 7.1 Implement `POST /api/nodes/{path}/refresh-description` using the same classification and digest generation flow as ingestion.
- [ ] 7.2 Preserve stable fields (`created`, `source_url`, `manual_processed`) and update descriptive fields (`annotation`, `keywords`, `source_kind`, `content_profile`, digest body) on successful refresh.
- [ ] 7.3 Allow refresh to correct `type` when classification changes an existing node from link to note or equivalent.
- [ ] 7.4 Trigger single-node reindex after successful refresh.
- [ ] 7.5 Add API tests for successful repository refresh, conceptual article refresh, news link correction, missing `source_url`, unknown path, and fetch/LLM failure without file mutation.
- [ ] 7.6 Add Action Bar button «Обновить описание из источника» for nodes with `source_url`.
- [ ] 7.7 Add UI loading, success, and error states for refresh-description.
- [ ] 7.8 Update UI state from the returned node, including changed `type` and available actions.
- [ ] 7.9 Add frontend tests or component coverage for button visibility, loading state, success update, and error display.

## 8. Verification

- [ ] 8.1 Run Go tests for `internal/kb`, `internal/ingestion`, `internal/index`, and `internal/api`.
- [ ] 8.2 Run frontend tests/build for `web`.
- [ ] 8.3 Run full backend test suite if targeted tests pass.
- [ ] 8.4 Validate OpenSpec status for `add-link-profile-digests`.
