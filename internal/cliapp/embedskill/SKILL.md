---
name: knowledge-db
description: Knowledge base layout and node format for knowledge-db. Use when creating or editing KB markdown files. Root path placeholder {{DATA_PATH}}.
---

# Knowledge base — knowledge-db

Use this skill when working with the user's knowledge base files (not application source).

## Base path

Knowledge base root: **{{DATA_PATH}}**

In the running app this is `KB_DATA_PATH`. Do not assume `./data` unless the user specifies it.

User-facing node content is often **Russian**; keep the existing language unless asked to translate.

## Node layout

Topics are directories (typically 2–3 levels deep). Each **node** is a single markdown file in a topic directory:

| Path | Purpose |
|------|---------|
| `{theme}/{slug}.md` | Main file: YAML frontmatter + markdown body |
| `{theme}/{slug}/` | Optional attachments (`images/`, `notes/`) — not a subtopic if `{slug}.md` exists |
| `{theme}/{slug}/.local/` | Local-only data (gitignored) |

There is **no** `{slug}/{slug}.md` folder-per-node layout.

## Frontmatter

Required fields on every main node file:

| Field | Description |
|-------|-------------|
| `id` | UUID v7 (lowercase), stable across move/rename |
| `keywords` | Array of tags for search |
| `created` | ISO 8601 — when added to the KB |
| `updated` | ISO 8601 — last content update |
| `type` | `article` \| `link` \| `note` |
| `title` | Human-readable title (not the slug) |

Common optional fields: `annotation`, `source_url`, `source_date`, `source_author`, `aliases`, `labels`, `manual_processed` (boolean).

Example:

```yaml
---
id: "01900000-0000-7000-8000-000000000001"
keywords: [example, go]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: "Example node"
annotation: "Short summary for lists and search"
---

# Example node

Body (markdown)…
```

## Topic hierarchy

- Organize as `topic/subtopic/…` directories.
- Place `{slug}.md` files inside the target topic directory.
- Do not exceed ~3 levels of topic nesting (validator may warn on deeper trees).

## Creating a node

1. Pick or create the topic path, e.g. `example/topic/`.
2. Add `{slug}.md` with valid frontmatter and body.
3. Optionally add `{slug}/images/` or `{slug}/notes/`.
4. Run validation (below).

Programmatic creation should assign a new UUID v7 when `id` is missing.

## Translation files

Translations live beside the original: `{slug}.{lang}.md` (e.g. `article.ru.md`).

- Translation frontmatter must include `translation_of`, `lang`, and its own `id`.
- The original should list `translations` and cross-link via wikilinks when a translation exists.
- See OpenSpec `article-translation` / `kb validate` rules for wikilink and field checks.

## Validation

From the machine where `kb` is installed:

```bash
kb validate --path "{{DATA_PATH}}"
```

Fix all reported errors before considering the node done.

## Normalization

When cleaning imported or scraped markdown in the KB repo, follow `.cursor/rules/markdown-normalization.mdc` if present in that repository.

Follow OpenSpec `knowledge-storage` when in doubt about storage rules.
